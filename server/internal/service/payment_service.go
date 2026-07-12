package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// EpayService handles 易支付 (Epay) payment gateway
type EpayService struct {
	URL       string
	PID       string
	Key       string
	NotifyURL string
	ReturnURL string
}

func NewEpayService(url, pid, key, notifyURL, returnURL string) *EpayService {
	return &EpayService{URL: url, PID: pid, Key: key, NotifyURL: notifyURL, ReturnURL: returnURL}
}

type CreateEpayOrderParams struct {
	OutTradeNo string
	Amount     float64
	Method     string // alipay, wxpay, qqpay
	Name       string
}

func (s *EpayService) CreateOrder(params CreateEpayOrderParams) string {
	values := url.Values{}
	values.Set("pid", s.PID)
	values.Set("type", params.Method)
	values.Set("out_trade_no", params.OutTradeNo)
	values.Set("notify_url", s.NotifyURL)
	values.Set("return_url", s.ReturnURL)
	values.Set("name", params.Name)
	values.Set("money", fmt.Sprintf("%.2f", params.Amount))

	sign := s.sign(values)
	values.Set("sign", sign)
	values.Set("sign_type", "MD5")

	return fmt.Sprintf("%s/submit.php?%s", strings.TrimRight(s.URL, "/"), values.Encode())
}

func (s *EpayService) VerifyCallback(values url.Values) bool {
	sign := values.Get("sign")
	values.Del("sign")
	values.Del("sign_type")

	calculated := s.sign(values)
	return strings.EqualFold(calculated, sign)
}

func (s *EpayService) sign(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if values.Get(k) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteString("&")
		}
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(values.Get(k))
	}
	buf.WriteString(s.Key)

	hash := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(hash[:])
}

// CodePayService handles 码支付 (CodePay) payment gateway
type CodePayService struct {
	URL       string
	ID        string
	Key       string
	NotifyURL string
	ReturnURL string
}

func NewCodePayService(url, id, key, notifyURL, returnURL string) *CodePayService {
	return &CodePayService{URL: url, ID: id, Key: key, NotifyURL: notifyURL, ReturnURL: returnURL}
}

type CreateCodePayOrderParams struct {
	OutTradeNo string
	Amount     float64
	Method     int // 1=alipay, 2=wechat, 3=qq
	Name       string
}

func (s *CodePayService) CreateOrder(params CreateCodePayOrderParams) string {
	values := url.Values{}
	values.Set("id", s.ID)
	values.Set("pay_id", params.OutTradeNo)
	values.Set("type", fmt.Sprintf("%d", params.Method))
	values.Set("price", fmt.Sprintf("%.2f", params.Amount))
	values.Set("notify_url", s.NotifyURL)
	values.Set("return_url", s.ReturnURL)
	values.Set("name", params.Name)

	sign := s.sign(values)
	values.Set("sign", sign)

	return fmt.Sprintf("%s/create_order?%s", strings.TrimRight(s.URL, "/"), values.Encode())
}

func (s *CodePayService) VerifyCallback(values url.Values) bool {
	sign := values.Get("sign")

	keys := []string{"id", "pay_id", "type", "price", "pay_no", "param", "pay_time"}
	var buf strings.Builder
	for _, k := range keys {
		v := values.Get(k)
		if v != "" {
			buf.WriteString(v)
		}
	}
	buf.WriteString(s.Key)

	hash := md5.Sum([]byte(buf.String()))
	calculated := hex.EncodeToString(hash[:])
	return strings.EqualFold(calculated, sign)
}

func (s *CodePayService) sign(values url.Values) string {
	keys := []string{"id", "pay_id", "type", "price", "notify_url", "return_url"}
	var buf strings.Builder
	for _, k := range keys {
		v := values.Get(k)
		if v != "" {
			buf.WriteString(v)
		}
	}
	buf.WriteString(s.Key)

	hash := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(hash[:])
}
