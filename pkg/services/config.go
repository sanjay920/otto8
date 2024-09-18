package services

import (
	"context"
	"os"

	"github.com/acorn-io/baaah"
	"github.com/acorn-io/baaah/pkg/leader"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/gptscript-ai/go-gptscript"
	"github.com/gptscript-ai/otto/pkg/aihelper"
	"github.com/gptscript-ai/otto/pkg/api"
	"github.com/gptscript-ai/otto/pkg/invoke"
	"github.com/gptscript-ai/otto/pkg/jwt"
	"github.com/gptscript-ai/otto/pkg/storage"
	"github.com/gptscript-ai/otto/pkg/storage/scheme"
	"github.com/gptscript-ai/otto/pkg/storage/services"
	"github.com/gptscript-ai/otto/pkg/system"
	wclient "github.com/thedadams/workspace-provider/pkg/client"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SystemToolKnowledge = "knowledge"
	SystemToolOneDrive  = "onedrive"
)

type Config struct {
	HTTPListenPort int    `usage:"HTTP port to listen on" default:"8080" name:"http-listen-port"`
	DevMode        bool   `usage:"Enable development mode" default:"false" name:"dev-mode" env:"OTTO_DEV_MODE"`
	AllowedOrigin  string `usage:"Allowed origin for CORS"`
	services.Config
}

type Services struct {
	StorageClient   storage.Client
	Router          *router.Router
	GPTClient       *gptscript.GPTScript
	Invoker         *invoke.Invoker
	TokenServer     *jwt.TokenService
	APIServer       *api.Server
	WorkspaceClient *wclient.Client
	AIHelper        *aihelper.AIHelper
	SystemTools     map[string]string
}

func New(ctx context.Context, config Config) (*Services, error) {
	system.SetBinToSelf()

	config = configureDevMode(config)

	storageClient, restConfig, err := storage.Start(ctx, config.Config)
	if err != nil {
		return nil, err
	}

	if config.DevMode {
		startDevMode(ctx, storageClient)
	}

	c, err := gptscript.NewGPTScript()
	if err != nil {
		return nil, err
	}

	r, err := baaah.NewRouter("otto-controller", &baaah.Options{
		DefaultRESTConfig: restConfig,
		Scheme:            scheme.Scheme,
		ElectionConfig:    leader.NewDefaultElectionConfig("", "otto-controller", restConfig),
	})
	if err != nil {
		return nil, err
	}

	tokenServer := &jwt.TokenService{}

	workspaceClient := wclient.New()

	return &Services{
		StorageClient:   storageClient,
		Router:          r,
		GPTClient:       c,
		APIServer:       api.NewServer(storageClient, c, tokenServer),
		TokenServer:     tokenServer,
		WorkspaceClient: workspaceClient,
		Invoker:         invoke.NewInvoker(storageClient, c, tokenServer, workspaceClient, config.KnowledgeTool),
		SystemTools: map[string]string{
			SystemToolKnowledge: config.KnowledgeTool,
			SystemToolOneDrive:  config.OneDriveTool,
		},
		AIHelper: aihelper.New(c, config.HelperModel),
	}, nil
}

func configureDevMode(config Config) Config {
	if !config.DevMode {
		return config
	}

	if config.StorageListenPort == 0 {
		if config.HTTPListenPort == 8080 {
			config.StorageListenPort = 8443
		} else {
			config.StorageListenPort = config.HTTPListenPort + 1
		}
	}
	if config.StorageToken == "" {
		config.StorageToken = "adminpass"
	}
	_ = os.Setenv("BAAAH_DEV_MODE", "true")

	return config
}

func startDevMode(ctx context.Context, storageClient storage.Client) {
	_ = storageClient.Delete(ctx, &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otto-controller",
			Namespace: "kube-system",
		},
	})
}
