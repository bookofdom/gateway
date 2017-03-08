package push

import (
	"crypto/tls"
	"encoding/json"
	"sync"

	"gateway/logreport"
	"gateway/model"
	re "gateway/model/remote_endpoint"

	apns "github.com/nanoscaleio/apns2"
	"github.com/nanoscaleio/apns2/certificate"
	"github.com/nanoscaleio/apns2/token"
	"github.com/vincent-petithory/dataurl"
)

type ApplePusher struct {
	sync.Mutex
	connection *apns.Client
	topic      string
	limit      bool
}

func NewApplePusher(platform *re.PushPlatform) *ApplePusher {
	if platform.TokenAuthentication {
		dataURL, err := dataurl.DecodeString(platform.AuthenticationKey)
		if err != nil {
			logreport.Fatal(err)
		}
		authKey, err := token.AuthKeyFromBytes(dataURL.Data)
		if err != nil {
			logreport.Fatal(err)
		}
		token := &token.Token{
			AuthKey: authKey,
			KeyID:   platform.KeyID,
			TeamID:  platform.TeamID,
		}
		client := apns.NewTokenClient(token)
		if platform.Development {
			client = client.Development()
		} else {
			client = client.Production()
		}
		return &ApplePusher{
			connection: client,
			topic:      platform.Topic,
			limit:      true,
		}
	}

	var cert tls.Certificate
	dataURL, err := dataurl.DecodeString(platform.Certificate)
	if err != nil {
		logreport.Fatal(err)
	}
	switch dataURL.MediaType.ContentType() {
	case re.PushCertificateTypePKCS12:
		cert, err = certificate.FromP12Bytes(dataURL.Data, platform.Password)
		if err != nil {
			logreport.Fatal(err)
		}
	case re.PushCertificateTypeX509:
		cert, err = certificate.FromPemBytes(dataURL.Data, platform.Password)
		if err != nil {
			logreport.Fatal(err)
		}
	default:
		logreport.Fatal("invalid apple certificate type")
	}
	client := apns.NewClient(cert)
	if platform.Development {
		client = client.Development()
	} else {
		client = client.Production()
	}
	return &ApplePusher{
		connection: client,
		topic:      platform.Topic,
	}
}

func (p *ApplePusher) Push(channel *model.PushChannel, device *model.PushDevice, data interface{}) error {
	if p.limit {
		p.Lock()
		defer p.Unlock()
	}
	notification := &apns.Notification{}
	notification.DeviceToken = device.Token
	notification.Topic = p.topic
	payload, err := json.Marshal(data)
	if err != nil {
		logreport.Fatal(err)
	}
	notification.Payload = payload
	_, err = p.connection.Push(notification)
	return err
}
