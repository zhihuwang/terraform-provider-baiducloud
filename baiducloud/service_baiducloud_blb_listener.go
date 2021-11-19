package baiducloud

import (
	"fmt"

	"github.com/baidubce/bce-sdk-go/services/blb"
)

func (s *BLBService) DescribeListener(blbId, protocol string, port int) (interface{}, error) {
	args := &blb.DescribeListenerArgs{
		ListenerPort: uint16(port),
	}
	action := fmt.Sprintf("Describe blb %s Listener [%s.%d]", blbId, protocol, port)

	raw, err := s.client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
		switch protocol {
		case TCP:
			return client.DescribeTCPListeners(blbId, args)
		case UDP:
			return client.DescribeUDPListeners(blbId, args)
		case HTTP:
			return client.DescribeHTTPListeners(blbId, args)
		case HTTPS:
			return client.DescribeHTTPSListeners(blbId, args)
		case SSL:
			return client.DescribeSSLListeners(blbId, args)
		default:
			return nil, fmt.Errorf("unsupport listener type: %s", protocol)
		}
	})
	addDebug(action, raw)

	if err != nil {
		return nil, WrapError(err)
	}

	switch protocol {
	case TCP:
		response := raw.(*blb.DescribeTCPListenersResult)
		if len(response.ListenerList) > 0 {
			return &response.ListenerList[0], nil
		}
	case UDP:
		response := raw.(*blb.DescribeUDPListenersResult)
		if len(response.ListenerList) > 0 {
			return &response.ListenerList[0], nil
		}
	case HTTP:
		response := raw.(*blb.DescribeHTTPListenersResult)
		if len(response.ListenerList) > 0 {
			return &response.ListenerList[0], nil
		}
	case HTTPS:
		response := raw.(*blb.DescribeHTTPSListenersResult)
		if len(response.ListenerList) > 0 {
			return &response.ListenerList[0], nil
		}
	case SSL:
		response := raw.(*blb.DescribeSSLListenersResult)
		if len(response.ListenerList) > 0 {
			return &response.ListenerList[0], nil
		}
	default:
	}

	return nil, WrapError(fmt.Errorf(ResourceNotFound))
}

func (s *BLBService) ListAllTCPListeners(blbId string, port int) ([]map[string]interface{}, error) {
	args := &blb.DescribeListenerArgs{
		ListenerPort: uint16(port),
	}

	action := fmt.Sprintf("Describe blb %s Listeners [TCP.%d]", blbId, port)
	listeners := make([]blb.TCPListenerModel, 0)
	for {
		raw, err := s.client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return client.DescribeTCPListeners(blbId, args)
		})
		if err != nil {
			return nil, WrapError(err)
		}
		addDebug(action, raw)

		response := raw.(*blb.DescribeTCPListenersResult)
		listeners = append(listeners, response.ListenerList...)

		if response.IsTruncated {
			args.Marker = response.Marker
			args.MaxKeys = response.MaxKeys
		} else {
			break
		}
	}

	result := make([]map[string]interface{}, 0, len(listeners))
	for _, listener := range listeners {
		result = append(result, map[string]interface{}{
			"listener_port":       listener.ListenerPort,
			"tcp_session_timeout": listener.TcpSessionTimeout,
			"protocol":            TCP,
			"scheduler":           listener.Scheduler,
		})
	}

	return result, nil
}

func (s *BLBService) ListAllUDPListeners(blbId string, port int) ([]map[string]interface{}, error) {
	args := &blb.DescribeListenerArgs{
		ListenerPort: uint16(port),
	}

	action := fmt.Sprintf("Describe blb %s Listeners [UDP.%d]", blbId, port)
	listeners := make([]blb.UDPListenerModel, 0)
	for {
		raw, err := s.client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return client.DescribeUDPListeners(blbId, args)
		})
		if err != nil {
			return nil, WrapError(err)
		}
		addDebug(action, raw)

		response := raw.(*blb.DescribeUDPListenersResult)
		listeners = append(listeners, response.ListenerList...)

		if response.IsTruncated {
			args.Marker = response.Marker
			args.MaxKeys = response.MaxKeys
		} else {
			break
		}
	}

	result := make([]map[string]interface{}, 0, len(listeners))
	for _, listener := range listeners {
		result = append(result, map[string]interface{}{
			"listener_port": listener.ListenerPort,
			"protocol":      UDP,
			"scheduler":     listener.Scheduler,
		})
	}

	return result, nil
}

func (s *BLBService) ListAllHTTPListeners(blbId string, port int) ([]map[string]interface{}, error) {
	args := &blb.DescribeListenerArgs{
		ListenerPort: uint16(port),
	}

	action := fmt.Sprintf("Describe blb %s Listeners [HTTP.%d]", blbId, port)
	listeners := make([]blb.HTTPListenerModel, 0)
	for {
		raw, err := s.client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return client.DescribeHTTPListeners(blbId, args)
		})
		if err != nil {
			return nil, WrapError(err)
		}
		addDebug(action, raw)

		response := raw.(*blb.DescribeHTTPListenersResult)
		listeners = append(listeners, response.ListenerList...)

		if response.IsTruncated {
			args.Marker = response.Marker
			args.MaxKeys = response.MaxKeys
		} else {
			break
		}
	}

	result := make([]map[string]interface{}, 0, len(listeners))
	for _, listener := range listeners {

		result = append(result, map[string]interface{}{
			"listener_port":            listener.ListenerPort,
			"protocol":                 HTTP,
			"scheduler":                listener.Scheduler,
			"keep_session":             listener.KeepSession,
			"keep_session_type":        listener.KeepSessionType,
			"keep_session_timeout":     listener.KeepSessionDuration,
			"keep_session_cookie_name": listener.KeepSessionCookieName,
			"x_forwarded_for":          listener.XForwardedFor,
			"server_timeout":           listener.ServerTimeout,
			"redirect_port":            listener.RedirectPort,
		})
	}

	return result, nil
}

func (s *BLBService) ListAllHTTPSListeners(blbId string, port int) ([]map[string]interface{}, error) {
	args := &blb.DescribeListenerArgs{
		ListenerPort: uint16(port),
	}

	action := fmt.Sprintf("Describe blb %s Listeners [HTTPS.%d]", blbId, port)
	listeners := make([]blb.HTTPSListenerModel, 0)
	for {
		raw, err := s.client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return client.DescribeHTTPSListeners(blbId, args)
		})
		if err != nil {
			return nil, WrapError(err)
		}
		addDebug(action, raw)

		response := raw.(*blb.DescribeHTTPSListenersResult)
		listeners = append(listeners, response.ListenerList...)

		if response.IsTruncated {
			args.Marker = response.Marker
			args.MaxKeys = response.MaxKeys
		} else {
			break
		}
	}

	result := make([]map[string]interface{}, 0, len(listeners))
	for _, listener := range listeners {

		result = append(result, map[string]interface{}{
			"listener_port":            listener.ListenerPort,
			"protocol":                 HTTPS,
			"scheduler":                listener.Scheduler,
			"keep_session":             listener.KeepSession,
			"keep_session_type":        listener.KeepSessionType,
			"keep_session_timeout":     listener.KeepSessionDuration,
			"keep_session_cookie_name": listener.KeepSessionCookieName,
			"x_forwarded_for":          listener.XForwardedFor,
			"server_timeout":           listener.ServerTimeout,
			"cert_ids":                 listener.CertIds,
			"dual_auth":                listener.DualAuth,
			"client_cert_ids":          listener.ClientCertIds,
		})
	}

	return result, nil
}

func (s *BLBService) ListAllSSLListeners(blbId string, port int) ([]map[string]interface{}, error) {
	args := &blb.DescribeListenerArgs{
		ListenerPort: uint16(port),
	}

	action := fmt.Sprintf("Describe blb %s Listeners [SSL.%d]", blbId, port)
	listeners := make([]blb.SSLListenerModel, 0)
	for {
		raw, err := s.client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return client.DescribeSSLListeners(blbId, args)
		})
		if err != nil {
			return nil, WrapError(err)
		}
		addDebug(action, raw)

		response := raw.(*blb.DescribeSSLListenersResult)
		listeners = append(listeners, response.ListenerList...)

		if response.IsTruncated {
			args.Marker = response.Marker
			args.MaxKeys = response.MaxKeys
		} else {
			break
		}
	}

	result := make([]map[string]interface{}, 0, len(listeners))
	for _, listener := range listeners {

		result = append(result, map[string]interface{}{
			"listener_port":        listener.ListenerPort,
			"protocol":             SSL,
			"scheduler":            listener.Scheduler,
			"cert_ids":             listener.CertIds,
			"encryption_type":      listener.EncryptionType,
			"encryption_protocols": listener.EncryptionProtocols,
			"dual_auth":            listener.DualAuth,
			"client_cert_ids":      listener.ClientCertIds,
		})
	}

	return result, nil
}

func (s *BLBService) ListAllListeners(blbId, protocol string, port int) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0)

	if protocol == TCP || protocol == "" {
		tcpListeners, err := s.ListAllTCPListeners(blbId, port)
		if err != nil {
			return nil, WrapError(err)
		}
		result = append(result, tcpListeners...)
	}

	if protocol == UDP || protocol == "" {
		udpListeners, err := s.ListAllUDPListeners(blbId, port)
		if err != nil {
			return nil, WrapError(err)
		}
		result = append(result, udpListeners...)
	}

	if protocol == HTTP || protocol == "" {
		httpListeners, err := s.ListAllHTTPListeners(blbId, port)
		if err != nil {
			return nil, WrapError(err)
		}
		result = append(result, httpListeners...)
	}

	if protocol == HTTPS || protocol == "" {
		httpsListeners, err := s.ListAllHTTPSListeners(blbId, port)
		if err != nil {
			return nil, WrapError(err)
		}
		result = append(result, httpsListeners...)
	}

	if protocol == SSL || protocol == "" {
		sslListeners, err := s.ListAllSSLListeners(blbId, port)
		if err != nil {
			return nil, WrapError(err)
		}
		result = append(result, sslListeners...)
	}

	return result, nil
}
