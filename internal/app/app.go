package app

import (
	"context"
	"fmt"
	"github.com/KolesnikNV/WireguardGo/internal/config"
	"github.com/KolesnikNV/WireguardGo/internal/logger"
	"github.com/KolesnikNV/WireguardGo/internal/proto/gen_proto"
	"github.com/KolesnikNV/WireguardGo/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"os"
	"os/signal"
	"syscall"
)

func MustStartApp() {
	cfg := config.MustLoad()
	log := logger.NewLogger(cfg.Env)
	grpcServer := grpc.NewServer()
	gen_proto.RegisterWireguardServer(grpcServer, &service.Wireguard{})
	reflection.Register(grpcServer)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	wireguard := &service.Wireguard{Logger: log}

	IP, PASSWORD := config.LoadServerData()

	_, err := wireguard.Connect(context.Background(), &gen_proto.ConnectResponse{
		IP:       IP,
		Password: PASSWORD,
	})
	if err != nil {
		log.Error("Failed to connect to Wireguard server: %v", err)
	}

	a, err := wireguard.GetConfig(context.Background(), &gen_proto.ConfigID{ConfId: "adb8065a-0ba9-43ff-8db1-345f06ce3e6a"})
	fmt.Println(a, err)

	<-stop

	log.Info("Gracefully stopped")
}
