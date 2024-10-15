package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/KolesnikNV/WireguardGo/internal/proto/gen_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"log/slog"
	"net/http"
)

// SessionClient содержит необходимые поля для авторизации на сервере и сохранения cookie
type SessionClient struct {
	IP       string
	Password string
	Client   *http.Client
}

type ConfigResponse struct {
	Name         string `json:"name"`
	Address      string `json:"address"`
	PrivateKey   string `json:"privateKey"`
	PublicKey    string `json:"publicKey"`
	PreSharedKey string `json:"preSharedKey"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	Enabled      bool   `json:"enabled"`
}

// Wireguard содержит методы для взаимодействия с Wireguard API
type Wireguard struct {
	gen_proto.UnimplementedWireguardServer
	session *SessionClient
	Logger  *slog.Logger
}

// Connect создаёт клиент для взаимодействия с сервером Wireguard
func (wg *Wireguard) Connect(ctx context.Context, req *gen_proto.ConnectResponse) (*emptypb.Empty, error) {
	fmt.Println("Attempting to connect to Wireguard server...")

	client, err := NewSessionClient(req.IP, req.Password)
	if err != nil {
		wg.Logger.Error("failed to create session client: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create session client: %v", err)
	}

	err = client.CreateSession()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}

	wg.session = client

	fmt.Println("Connected to Wireguard server successfully.")

	return &emptypb.Empty{}, nil
}

// AddConfig создает новый конфиг на сервере Wireguard
func (wg *Wireguard) AddConfig(ctx context.Context, res *gen_proto.ConfigName) (*gen_proto.AddConfigResponse, error) {

	resp, err := wg.PrepareAndSendPOSTRequest("api/wireguard/client", map[string]string{"name": res.ConfName})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create request: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	defer resp.Body.Close()

	var configResponse ConfigResponse
	if err := json.Unmarshal(body, &configResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	configAddress := &gen_proto.ConfigAddress{
		ConfAddress: configResponse.Address,
	}
	idResponse, err := wg.GetConfigID(ctx, &gen_proto.ConfigAddress{ConfAddress: configResponse.Address})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get config ID: %v", err)
	}
	fmt.Println(idResponse)
	configID := &gen_proto.ConfigID{
		ConfId: "",
	}

	addConfigResponse := &gen_proto.AddConfigResponse{
		ConfigId:      configID,
		ConfigAddress: configAddress,
	}

	return addConfigResponse, nil
}

// GetConfig получает конфигурацию по ID
func (wg *Wireguard) GetConfig(ctx context.Context, req *gen_proto.ConfigID) (*gen_proto.ConfigText, error) {
	url := fmt.Sprintf("api/wireguard/client/%s/configuration", req.ConfId)
	resp, err := wg.PrepareAndSendGETRequest(url)
	if err != nil {
		wg.Logger.Error("failed to create request: ", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to create request: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		wg.Logger.Error("failed to read response body: ", "err", err)
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return &gen_proto.ConfigText{Text: string(body)}, nil
}

// GetAllConfigs получает все конфигурации
func (wg *Wireguard) GetAllConfigs(ctx context.Context, _ *emptypb.Empty) (*gen_proto.GetAllConfigsResponse, error) {
	resp, err := wg.PrepareAndSendGETRequest("api/wireguard/client")
	defer resp.Body.Close()
	if err != nil {
		wg.Logger.Error("failed to create request: ", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to create request: %v", err)
	}

	var tempResponse []struct {
		ID                  string `json:"id"`
		Name                string `json:"name"`
		Enabled             bool   `json:"enabled"`
		Address             string `json:"address"`
		PublicKey           string `json:"publicKey"`
		CreatedAt           string `json:"createdAt"`
		UpdatedAt           string `json:"updatedAt"`
		PersistentKeepalive string `json:"persistentKeepalive"`
		LatestHandshakeAt   string `json:"latestHandshakeAt"`
		TransferRx          int64  `json:"transferRx"`
		TransferTx          int64  `json:"transferTx"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		wg.Logger.Error("failed to read response body: ", "err", err)
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &tempResponse)
	if err != nil {
		wg.Logger.Error("failed to unmarshal response body: ", "err", err)
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	var protoResponse gen_proto.GetAllConfigsResponse
	for _, cfg := range tempResponse {
		protoResponse.ConfList = append(protoResponse.ConfList, &gen_proto.GetConfigResponse{
			Id:                  cfg.ID,
			Name:                cfg.Name,
			Enabled:             cfg.Enabled,
			Address:             cfg.Address,
			PublicKey:           cfg.PublicKey,
			CreatedAt:           cfg.CreatedAt,
			UpdatedAt:           cfg.UpdatedAt,
			PersistentKeepalive: cfg.PersistentKeepalive,
			LatestHandshakeAt:   cfg.LatestHandshakeAt,
			TransferRx:          cfg.TransferRx,
			TransferTx:          cfg.TransferTx,
		})
	}

	return &protoResponse, nil
}

// GetConfigID получает конфигурацию по ID
func (wg *Wireguard) GetConfigID(ctx context.Context, res *gen_proto.ConfigAddress) (*gen_proto.ConfigIdResponse, error) {
	data, err := wg.GetAllConfigs(ctx, &emptypb.Empty{})
	if err != nil {
		wg.Logger.Error("failed to get configs: ", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to get configs: %v", err)
	}

	for _, config := range data.ConfList {
		if config.Address == res.ConfAddress {
			return &gen_proto.ConfigIdResponse{ConfigId: &gen_proto.ConfigID{ConfId: config.Id}}, nil
		}

	}
	return &gen_proto.ConfigIdResponse{}, status.Errorf(codes.Internal, "failed to get config id")

}

// GetConfigsAmount возвращает количество конфигов на сервере
func (wg *Wireguard) GetConfigsAmount(ctx context.Context, res *emptypb.Empty) (*gen_proto.ConfigAmount, error) {
	data, err := wg.GetAllConfigs(context.Background(), &emptypb.Empty{})
	if err != nil {
		wg.Logger.Error("failed to get configs: ", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to get configs: %v", err)
	}

	return &gen_proto.ConfigAmount{Amount: int32(len(data.ConfList))}, nil
}

// EnableConfig включает конфигурацию на сервере
func (wg *Wireguard) EnableConfig(ctx context.Context, res *gen_proto.ConfigID) (*emptypb.Empty, error) {
	url := fmt.Sprintf("api/wireguard/client/%s/enable", res.ConfId)
	_, err := wg.PrepareAndSendPOSTRequest(url, nil)
	if err != nil {
		wg.Logger.Error("failed to create request: ", err)
		return nil, status.Errorf(codes.Internal, "failed to create request: %v", err)
	}

	return &emptypb.Empty{}, nil

}

// DisableConfig выключает конфигурацию на сервере
func (wg *Wireguard) DisableConfig(ctx context.Context, res *gen_proto.ConfigID) (*emptypb.Empty, error) {
	url := fmt.Sprintf("api/wireguard/client/%s/disable", res.ConfId)
	_, err := wg.PrepareAndSendPOSTRequest(url, nil)
	if err != nil {
		wg.Logger.Error("failed to create request: ", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to create request: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// DeleteConfig удаляет конфигурацию с сервера
func (wg *Wireguard) DeleteConfig(ctx context.Context, res *gen_proto.ConfigID) (*emptypb.Empty, error) {
	url := fmt.Sprintf("api/wireguard/client/%s", res.ConfId)
	_, err := wg.PrepareAndSendPOSTRequest(url, nil)
	if err != nil {
		wg.Logger.Error("failed to create request: ", "err", err)

		return nil, status.Errorf(codes.Internal, "failed to create request: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetConfig возвращает конфигурацию с сервера в виде QR-код
func (wg *Wireguard) GetQR(ctx context.Context, res *gen_proto.ConfigID) (*gen_proto.QRCode, error) {
	url := fmt.Sprintf("api/wireguard/client/%s/qrcode.svg", res.ConfId)

	resp, err := wg.PrepareAndSendGETRequest(url)
	if err != nil {
		wg.Logger.Error("failed to get QR code: %v", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to get QR code: %v", err)
	}
	defer resp.Body.Close()

	qrCodeData, err := io.ReadAll(resp.Body)
	if err != nil {
		wg.Logger.Error("failed to read QR code data: %v", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to read QR code data: %v", err)
	}

	qrCode := &gen_proto.QRCode{
		QrCode: qrCodeData,
	}

	return qrCode, nil
}
