package ftpserver

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
	ftpserverlib "github.com/fclairamb/ftpserverlib"
	"github.com/mmcdole/viking-ftpd/pkg/authentication"
	"github.com/mmcdole/viking-ftpd/pkg/authorization"
)

// Config holds FTP server configuration
type Config struct {
	ListenAddr            string
	Port                  int
	RootDir              string  // Root directory that FTP users will be restricted to
	HomePattern          string  // Pattern for user home directories (e.g., "/home/%s" where %s is username)
	PassiveTransferPorts  [2]int
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

// GetSettings returns server settings
func (d *ftpDriver) GetSettings() (*ftpserverlib.Settings, error) {
	return &ftpserverlib.Settings{
		ListenAddr: fmt.Sprintf("%s:%d", d.server.config.ListenAddr, d.server.config.Port),
		PassiveTransferPortRange: &ftpserverlib.PortRange{
			Start: d.server.config.PassiveTransferPorts[0],
			End:   d.server.config.PassiveTransferPorts[1],
		},
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
	// Authenticate user
	if err := d.server.authenticator.Authenticate(user, pass); err != nil {
		return nil, err
	}

	// Create home directory path
	homePath := ""
	if d.server.config.HomePattern != "" {
		homePath = fmt.Sprintf(d.server.config.HomePattern, user)
		homePath = filepath.Clean(homePath) // Clean to remove any .. or . components
		
		// Create full home directory path
		fullHomePath := filepath.Join(d.server.config.RootDir, homePath)
		
		// Create home directory if it doesn't exist
		if err := os.MkdirAll(fullHomePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create home directory: %w", err)
		}
	}

	// Create a new filesystem rooted at the server's root directory
	fs := afero.NewBasePathFs(afero.NewOsFs(), d.server.config.RootDir)
	
	// Set the initial path to their home directory (relative to root)
	if homePath != "" {
		cc.SetPath("/" + homePath)
	} else {
		cc.SetPath("/")
	}
	
	return &ftpClient{
		server:   d.server,
		user:     user,
		fs:       fs,
		homePath: homePath,
		rootPath: d.server.config.RootDir,
	}, nil
}

// GetTLSConfig returns TLS config
func (d *ftpDriver) GetTLSConfig() (*tls.Config, error) {
	return nil, nil
}

// ftpClient implements both ftpserverlib.ClientDriver and afero.Fs interfaces
type ftpClient struct {
	server   *Server
	user     string
	fs       afero.Fs
	homePath string // User's home directory path (relative to root)
	rootPath string // Server's root directory absolute path
}

// resolvePath ensures a path doesn't escape the server root
func (c *ftpClient) resolvePath(name string) (string, error) {
	// Clean the path to remove any ".." or "." components
	cleanPath := filepath.Clean(name)
	
	// If it's an absolute path, remove the leading /
	if filepath.IsAbs(cleanPath) {
		cleanPath = cleanPath[1:]
	}
	
	// Ensure the path doesn't escape the root directory
	fullPath := filepath.Join(c.rootPath, cleanPath)
	if !filepath.HasPrefix(fullPath, c.rootPath) {
		return "", os.ErrPermission
	}
	
	return cleanPath, nil
}

// GetFS returns the filesystem - part of ftpserverlib.ClientDriver interface
func (c *ftpClient) GetFS() afero.Fs {
	return c.fs
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
	
	return f.(interface{ Readdir(count int) ([]os.FileInfo, error) }).Readdir(-1)
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

// Create creates a new file - part of afero.Fs interface
func (c *ftpClient) Create(name string) (afero.File, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		return nil, err
	}
	
	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return nil, os.ErrPermission
	}
	return c.fs.Create(path)
}

// Mkdir creates a directory - part of afero.Fs interface
func (c *ftpClient) Mkdir(name string, perm os.FileMode) error {
	path, err := c.resolvePath(name)
	if err != nil {
		return err
	}
	
	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.Mkdir(name, perm)
}

// MkdirAll creates a directory and all parent directories - part of afero.Fs interface
func (c *ftpClient) MkdirAll(path string, perm os.FileMode) error {
	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return err
	}
	
	if !c.server.authorizer.GetEffectivePermission(c.user, resolvedPath).CanWrite() {
		return os.ErrPermission
	}
	return c.fs.MkdirAll(resolvedPath, perm)
}

// Open opens a file for reading - part of afero.Fs interface
func (c *ftpClient) Open(name string) (afero.File, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		return nil, err
	}
	
	if !c.server.authorizer.GetEffectivePermission(c.user, path).CanRead() {
		return nil, os.ErrPermission
	}
	return c.fs.Open(path)
}

// OpenFile opens a file using the given flags and mode - part of afero.Fs interface
func (c *ftpClient) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	path, err := c.resolvePath(name)
	if err != nil {
		return nil, err
	}
	
	p := c.server.authorizer.GetEffectivePermission(c.user, path)
	
	if flag&os.O_RDONLY != 0 && !p.CanRead() {
		return nil, os.ErrPermission
	}
	
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 && !p.CanWrite() {
		return nil, os.ErrPermission
	}
	
	return c.fs.OpenFile(path, flag, perm)
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
