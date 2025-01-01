package ftpserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ftpserverlib "github.com/fclairamb/ftpserverlib"
	"github.com/mmcdole/viking-ftpd/pkg/authentication"
	"github.com/mmcdole/viking-ftpd/pkg/authorization"
	"github.com/spf13/afero"
)

// Config holds FTP server configuration
type Config struct {
	ListenAddr           string
	Port                 int
	RootDir              string // Root directory that FTP users will be restricted to
	HomePattern          string // Pattern for user home directories (e.g., "/home/%s" where %s is username)
	PassiveTransferPorts [2]int
	TLSCertFile          string // Path to TLS certificate file
	TLSKeyFile           string // Path to TLS private key file
}

// Server wraps the FTP server with our custom auth
type Server struct {
	config        *Config
	authorizer    *authorization.Authorizer
	authenticator *authentication.Authenticator
	server        *ftpserverlib.FtpServer
}

// New creates a new FTP server
func New(config *Config, authorizer *authorization.Authorizer, authenticator *authentication.Authenticator) (*Server, error) {
	// Validate config
	if _, err := os.Stat(config.RootDir); err != nil {
		return nil, fmt.Errorf("root directory does not exist: %w", err)
	}

	s := &Server{
		config:        config,
		authorizer:    authorizer,
		authenticator: authenticator,
	}

	driver := &ftpDriver{server: s}
	s.server = ftpserverlib.NewFtpServer(driver)

	return s, nil
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Stop stops the server
func (s *Server) Stop() error {
	return s.server.Stop()
}

// ftpDriver implements ftpserverlib.MainDriver
type ftpDriver struct {
	server *Server
}

var errNoTLS = errors.New("TLS is not configured")

// GetSettings returns server settings
func (d *ftpDriver) GetSettings() (*ftpserverlib.Settings, error) {
	return &ftpserverlib.Settings{
		ListenAddr: fmt.Sprintf("%s:%d", d.server.config.ListenAddr, d.server.config.Port),
		PassiveTransferPortRange: &ftpserverlib.PortRange{
			Start: d.server.config.PassiveTransferPorts[0],
			End:   d.server.config.PassiveTransferPorts[1],
		},
		TLSRequired: ftpserverlib.ClearOrEncrypted,
	}, nil
}

// ClientConnected is called when a client connects
func (d *ftpDriver) ClientConnected(cc ftpserverlib.ClientContext) (string, error) {
	return "Welcome to Viking FTP server", nil
}

// ClientDisconnected is called when a client disconnects
func (d *ftpDriver) ClientDisconnected(cc ftpserverlib.ClientContext) {
	// Nothing to do
}

// AuthUser authenticates the user and returns a ClientDriver
func (d *ftpDriver) AuthUser(cc ftpserverlib.ClientContext, user, pass string) (ftpserverlib.ClientDriver, error) {
	fmt.Printf("AuthUser: authenticating user=%s\n", user)

	// Authenticate user
	if err := d.server.authenticator.Authenticate(user, pass); err != nil {
		fmt.Printf("AuthUser: authentication failed for user=%s: %v\n", user, err)
		return nil, err
	}
	fmt.Printf("AuthUser: authentication successful for user=%s\n", user)

	// Generate home directory path using proper path joining
	var homePath string
	if d.server.config.HomePattern != "" {
		// Create clean, relative path for home directory
		homePath = filepath.Clean(fmt.Sprintf(d.server.config.HomePattern, user))
		fmt.Printf("AuthUser: using home path: %s\n", homePath)
	}

	// Create filesystem with root already handled
	fs := afero.NewBasePathFs(afero.NewOsFs(), d.server.config.RootDir)

	// Set initial FTP path to home directory
	initialPath := filepath.Join("/", homePath)
	cc.SetPath(initialPath)

	client := &ftpClient{
		server:   d.server,
		user:     user,
		fs:       fs,
		homePath: homePath,
		rootPath: d.server.config.RootDir,
	}
	fmt.Printf("AuthUser: created client with homePath=%s, rootPath=%s\n", client.homePath, client.rootPath)
	return client, nil
}

// GetTLSConfig returns TLS config
func (d *ftpDriver) GetTLSConfig() (*tls.Config, error) {
	if d.server.config.TLSCertFile == "" || d.server.config.TLSKeyFile == "" {
		// If no TLS config is provided, return error to indicate no TLS support
		return nil, errNoTLS
	}

	cert, err := tls.LoadX509KeyPair(d.server.config.TLSCertFile, d.server.config.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("loading TLS cert/key pair: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// ftpClient implements both ftpserverlib.ClientDriver and afero.Fs interfaces
type ftpClient struct {
	server   *Server
	user     string
	fs       afero.Fs
	homePath string // User's home directory path (relative to root)
	rootPath string // Server's root directory absolute path
}

// resolvePath converts FTP protocol paths to filesystem paths by cleaning the path
// and converting absolute paths (with leading /) to relative paths
func (c *ftpClient) resolvePath(name string) (string, error) {
	fmt.Printf("resolvePath: input=%s, user=%s, homePath=%s, rootPath=%s\n", name, c.user, c.homePath, c.rootPath)

	// Convert FTP protocol path (always forward slashes) to system path
	cleanPath := filepath.FromSlash(filepath.Clean(name))

	// Root path is special
	if cleanPath == "/" || cleanPath == "." || cleanPath == "" {
		return "", nil
	}

	// Strip leading slash for relative paths
	if strings.HasPrefix(cleanPath, string(filepath.Separator)) {
		cleanPath = cleanPath[1:]
	}

	return cleanPath, nil
}

// GetFS returns the filesystem - part of ftpserverlib.ClientDriver interface
func (c *ftpClient) GetFS() afero.Fs {
	return c
}

// =====================================
// FTP Server-Specific Methods
// These are specific to ftpserverlib.ClientDriver and its extensions
// =====================================

// ReadDir is required by ftpserverlib for directory listing
func (c *ftpClient) ReadDir(name string) ([]os.FileInfo, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		return nil, err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanRead() {
		return nil, os.ErrPermission
	}

	f, err := c.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.(interface {
		Readdir(count int) ([]os.FileInfo, error)
	}).Readdir(-1)
}

// DeleteFile is required by ftpserverlib for DELE command
func (c *ftpClient) DeleteFile(name string) error {
	path, err := c.resolvePath(name)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Remove(path)
}

// MakeDirectory is required by ftpserverlib for MKD command
func (c *ftpClient) MakeDirectory(name string) error {
	path, err := c.resolvePath(name)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.MkdirAll(path, 0755)
}

// =====================================
// afero.Fs Interface Methods
// These implement the standard filesystem interface
// =====================================

// Open opens a file for reading - part of afero.Fs interface
func (c *ftpClient) Open(name string) (afero.File, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		fmt.Printf("Open: error resolving path %s: %v\n", name, err)
		return nil, err
	}

	perm := c.server.authorizer.GetEffectivePermission(c.user, path)
	fmt.Printf("Open: user=%s, path=%s, perm=%#v, canRead=%v\n", c.user, path, perm, perm.CanRead())
	if !perm.CanRead() {
		return nil, os.ErrPermission
	}
	return c.fs.Open(path)
}

// OpenFile opens a file using the given flags and mode - part of afero.Fs interface
func (c *ftpClient) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		fmt.Printf("OpenFile: error resolving path %s: %v\n", name, err)
		return nil, err
	}

	p := c.server.authorizer.GetEffectivePermission(c.user, path)
	fmt.Printf("OpenFile: user=%s, path=%s, perm=%#v, flag=%v, canRead=%v, canWrite=%v\n",
		c.user, path, p, flag, p.CanRead(), p.CanWrite())

	if flag&os.O_RDONLY != 0 && !p.CanRead() {
		return nil, os.ErrPermission
	}
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 && !p.CanWrite() {
		return nil, os.ErrPermission
	}

	return c.fs.OpenFile(path, flag, perm)
}

// Create creates a new file - part of afero.Fs interface
func (c *ftpClient) Create(name string) (afero.File, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		fmt.Printf("Create: error resolving path %s: %v\n", name, err)
		return nil, err
	}

	perm := c.server.authorizer.GetEffectivePermission(c.user, path)
	fmt.Printf("Create: user=%s, path=%s, perm=%#v, canWrite=%v\n", c.user, path, perm, perm.CanWrite())
	if !perm.CanWrite() {
		return nil, os.ErrPermission
	}
	return c.fs.Create(path)
}

// Mkdir creates a directory - part of afero.Fs interface
func (c *ftpClient) Mkdir(name string, perm os.FileMode) error {
	path, err := c.resolvePath(name)
	if err != nil {
		fmt.Printf("Mkdir: error resolving path %s: %v\n", name, err)
		return err
	}

	p := c.server.authorizer.GetEffectivePermission(c.user, path)
	fmt.Printf("Mkdir: user=%s, path=%s, perm=%#v, canWrite=%v\n", c.user, path, p, p.CanWrite())
	if !p.CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Mkdir(name, perm)
}

// MkdirAll creates a directory and all parent directories - part of afero.Fs interface
func (c *ftpClient) MkdirAll(path string, perm os.FileMode) error {
	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		fmt.Printf("MkdirAll: error resolving path %s: %v\n", path, err)
		return err
	}

	p := c.server.authorizer.GetEffectivePermission(c.user, resolvedPath)
	fmt.Printf("MkdirAll: user=%s, path=%s, perm=%#v, canWrite=%v\n", c.user, resolvedPath, p, p.CanWrite())
	if !p.CanWrite() {
		return os.ErrPermission
	}
	return c.fs.MkdirAll(resolvedPath, perm)
}

// Remove removes a file - part of afero.Fs interface
func (c *ftpClient) Remove(name string) error {
	path, err := c.resolvePath(name)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Remove(path)
}

// RemoveAll removes a directory and all its contents - part of afero.Fs interface
func (c *ftpClient) RemoveAll(path string) error {
	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, resolvedPath).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.RemoveAll(resolvedPath)
}

// Rename renames a file - part of afero.Fs interface
func (c *ftpClient) Rename(oldname, newname string) error {
	oldPath, err := c.resolvePath(oldname)
	if err != nil {
		return err
	}

	newPath, err := c.resolvePath(newname)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, oldPath).CanWrite() ||
		!c.server.authorizer.GetEffectivePermission(c.user, newPath).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Rename(oldPath, newPath)
}

// Stat returns file info - part of afero.Fs interface
func (c *ftpClient) Stat(name string) (os.FileInfo, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		return nil, err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanRead() {
		return nil, os.ErrPermission
	}
	return c.fs.Stat(path)
}

// Name returns the name of the filesystem - part of afero.Fs interface
func (c *ftpClient) Name() string {
	return "VikingFTPD"
}

// Chmod changes file mode - part of afero.Fs interface
func (c *ftpClient) Chmod(name string, mode os.FileMode) error {
	path, err := c.resolvePath(name)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Chmod(path, mode)
}

// Chown changes file owner - part of afero.Fs interface
func (c *ftpClient) Chown(name string, uid, gid int) error {
	path, err := c.resolvePath(name)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Chown(path, uid, gid)
}

// Chtimes changes file times - part of afero.Fs interface
func (c *ftpClient) Chtimes(name string, atime, mtime time.Time) error {
	path, err := c.resolvePath(name)
	if err != nil {
		return err
	}

	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Chtimes(path, atime, mtime)
}
