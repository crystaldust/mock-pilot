package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gogo/protobuf/types"
	"istio.io/istio/pilot/pkg/bootstrap"
	"istio.io/istio/pilot/pkg/proxy/envoy"
	"istio.io/istio/pkg/test/env"
	"k8s.io/apimachinery/pkg/util/wait"
)

func main() {
	// The code are yanked from Istio's test packages.
	if err := setup(); err != nil {
		panic("Failed to setup mock pilot server: " + err.Error())
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	close(stop)
}

var (
	// MockTestServer is used for the unit tests. Will be started once, terminated at the
	// end of the suite.
	MockTestServer *bootstrap.Server

	stop chan struct{}

	HttpPort int = 15011
	GrpcPort int = 15010
)

func setup() error {
	// TODO: point to test data directory
	// Setting FileDir (--configDir) disables k8s client initialization, including for registries,
	// and uses a 100ms scan. Must be used with the mock registry (or one of the others)
	// This limits the options -
	stop = make(chan struct{})

	args := prepareArgs()
	// Create a test pilot discovery service configured to watch the tempDir.
	// Create and setup the controller.
	s, err := bootstrap.NewServer(args)
	if err != nil {
		return err
	}

	MockTestServer = s

	// Start the server.
	if err := s.Start(stop); err != nil {
		return err
	}

	// Wait a bit for the server to come up.
	err = wait.Poll(500*time.Millisecond, 5*time.Second, func() (bool, error) {
		client := &http.Client{Timeout: 1 * time.Second}
		readyUrl := fmt.Sprintf("http://localhost:%d/ready", HttpPort)
		resp, err := client.Get(readyUrl)
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true, nil
		}
		return false, nil
	})
	return err
}

func prepareArgs() bootstrap.PilotArgs {
	args := bootstrap.PilotArgs{
		Namespace: "testing",
		DiscoveryOptions: envoy.DiscoveryServiceOptions{
			HTTPAddr:        fmt.Sprintf(":%d", HttpPort),
			GrpcAddr:        fmt.Sprintf(":%d", GrpcPort),
			EnableCaching:   true,
			EnableProfiling: true,
		},
		//TODO: start mixer first, get its address
		Mesh: bootstrap.MeshArgs{
			MixerAddress:    "istio-mixer.istio-system:9091",
			RdsRefreshDelay: types.DurationProto(10 * time.Millisecond),
		},
		Config: bootstrap.ConfigArgs{
			KubeConfig: env.IstioSrc + "/.circleci/config",
		},
		Service: bootstrap.ServiceArgs{
			// Using the Mock service registry, which provides the hello and world services.
			Registries: []string{"Mock"}, // Taken from istio.io/istio/pilot/pkg/serviceregistry.MockRegistry
		},
		MCPMaxMessageSize: bootstrap.DefaultMCPMaxMsgSize,
	}
	// Static testdata, should include all configs we want to test.
	args.Config.FileDir = "./tests/testdata/config"

	return args
}
