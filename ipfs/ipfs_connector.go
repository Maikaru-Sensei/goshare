package ipfs

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	_ "github.com/fatih/color"
	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Connector struct {
	Api      icore.CoreAPI
	Node     *core.IpfsNode
	RepoPath string
}

var loadPluginsOnce sync.Once

func loadAndInjectPlugins(externalPluginsPath string) error {
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func setupPlugins() error {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = loadAndInjectPlugins("")
	})
	if onceErr != nil {
		return onceErr
	}
	return nil
}

func initRepository(repository string) error {
	err := os.Mkdir(repository, 0777)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create/get repo dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		return err
	}

	// Create the repo with the config
	err = fsrepo.Init(repository, cfg)
	if err != nil {
		return fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return nil
}

func buildNode(ctx context.Context, repository string) (*core.IpfsNode, error) {
	repo, err := fsrepo.Open(repository)
	if err != nil {
		return nil, err
	}

	// build the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)

	return node, err
}

func CreateNode(ctx context.Context, repository string) (*Connector, error) {
	err := setupPlugins()
	if err != nil {
		return nil, err
	}

	err = initRepository(repository)
	if err != nil {
		return nil, err
	}

	node, err := buildNode(ctx, repository)
	if err != nil {
		return nil, err
	}

	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs api: %s", err)
	}

	return &Connector{Api: api, Node: node, RepoPath: repository}, err
}

func getFsFile(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (c *Connector) AddFile(ctx context.Context, filePath string) error {
	file, err := getFsFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load file: %s", err)
	}

	fileCid, err := c.Api.Unixfs().Add(ctx, file)
	if err != nil {
		return fmt.Errorf("failed to add file: %s", err)
	}

	color.Green("Added file with Cid: %s\n", fileCid.RootCid())

	return err
}

func (c *Connector) GetFile(ctx context.Context, contentId string, outputPath string) error {
	cidFile, _ := cid.Decode(contentId)
	file, err := c.Api.Unixfs().Get(ctx, path.FromCid(cidFile))
	if err != nil {
		return err
	}

	err = files.WriteTo(file, outputPath)
	if err != nil {
		return err
	}

	color.Green("Successfully Wrote file to %s", outputPath)
	return err
}
