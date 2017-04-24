package sms

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const SMSRU_API_URL = "http://sms.ru"

var codeStatus map[int]string = map[int]string{
	-1:  "Not found",
	100: "Success",
	101: "The messege is passed to operator",
	102: "The message sent (in transit)",
	103: "The message was delivered",
	104: "Cannot be delivered: Time of life expired",
	105: "Cannot be delivered: deleted by operator",
	106: "Cannot be delivered: phone failure",
	107: "Cannot be delivered: unknown reason",
	108: "Cannot be delivered: rejected",
	130: "Cannot be delivered: Daily message limit on this number was exceeded",
	131: "Cannot be delivered: Same messages limit on this phone number in a minute was exceeded",
	132: "Cannot be delivered: Same messages limit on this phone number in a day was exceeded",
	200: "Wrong api_id",
	201: "Too low balance",
	202: "Wrong recipient",
	203: "The message has no text",
	204: "Sender name did not approve with administartion",
	205: "The message is too long (more than 8 sms)",
	206: "Daily message limit exceeded",
	207: "On this phone number (or one of them) must not send the messages, or you indicated more than 100 phone numbers",
	208: "Wrong time value",
	209: "You added this phone number (or one of them) in the stop-list",
	210: "You must use a POST, not a GET",
	211: "Method not found",
	212: "Text of message must be in UTF-8",
	220: "The service is not availiable now, try again later",
	230: "Daily message limit on this number was exceeded",
	231: "Same messages limit on this phone number in a minute was exceeded",
	232: "Same messages limit on this phone number in a day was exceeded",
	300: "Wrong token (maybe it was expired or your IP was changed)",
	301: "Wrong password, or user is not exist",
	302: "User was authorized, but account is not activate",
	901: "Wrong Url (should begin with 'http://')",
	902: "Callback is not defined",
}

var error_internal = errors.New("Internal Error")
var error_no_response = errors.New("Something went wrong")


// id is your api_id
func NewSmsRuClient(apiId string, sender string) *SmsRuClient {

	c := &SmsRuClient{
		ApiId: apiId,
		Http:  &http.Client{},
		From:  sender,
	}

	return c
}


// NewSms creates a new message

func (c *SmsRuClient) NewSms(to string, text string) *Sms {
	return &Sms{
		To:   to,
		Text: text,
		From: c.From,
	}
}


func (c *SmsRuClient) makeRequest(endpoint string, params url.Values) (Response, []string, error) {
	params.Set("api_id", c.ApiId)
	url := SMSRU_API_URL + endpoint + "?" + params.Encode()

	resp, err := c.Http.Get(url)
	if err != nil {
		return Response{}, nil, err
	}
	defer resp.Body.Close()

	sc := bufio.NewScanner(resp.Body)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	if err := sc.Err(); err != nil {
		return Response{}, nil, error_internal
	}

	if len(lines) == 0 {
		return Response{}, nil, error_no_response
	}

	status, _ := strconv.Atoi(lines[0])

	if status >= 200 {
		msg := fmt.Sprintf("Code: %d; Status: %s", status, codeStatus[status])
		return Response{}, nil, errors.New(msg)
	}

	res := Response{Status: codeStatus[status]}
	return res, lines, nil
}

// SmsSend will send a Sms item to Service
func (c *SmsRuClient) SmsSend(p *Sms) (Response, error) {
	var params = url.Values{}

	if len(p.Multi) > 0 {
		for to, text := range p.Multi {
			key := fmt.Sprintf("multi[%s]", to)
			params.Add(key, text)
		}
	} else {
		params.Set("to", p.To)
		params.Set("text", p.Text)
	}

	if len(p.From) > 0 {
		params.Set("from", p.From)
	}

	if p.PartnerId > 0 {
		val := strconv.Itoa(p.PartnerId)
		params.Set("partner_id", val)
	}

	if p.Test {
		params.Set("test", "1")
	}

	if p.Time.After(time.Now()) {
		val := strconv.FormatInt(p.Time.Unix(), 10)
		params.Set("time", val)
	}

	if p.Translit {
		params.Set("translit", "1")
	}

	res, lines, err := c.makeRequest("/sms/send", params)
	if err != nil {
		return Response{}, err
	}

	res.Id = lines[1]
	res.Phone = p.To
	return res, nil
}

// SmsStatus will get a status of message
func (c *SmsRuClient) SmsStatus(id string) (Response, error) {
	params := url.Values{}
	params.Set("id", id)

	res, _, err := c.makeRequest("/sms/status", params)
	if err != nil {
		return Response{}, err
	}

	res.Id = id

	return res, nil
}
