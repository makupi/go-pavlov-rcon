package go_rcon

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	DefaultTimeout = 5.0 * time.Second
)

type GoRCONClientState string

const (
	GoRCONClientStateIdle      = GoRCONClientState("idle")
	GoRCONClientStateConnected = GoRCONClientState("connected")
	readLength                 = 256
)

type Client struct {
	conn          net.Conn
	address       string
	passwordHash  string
	State         GoRCONClientState
	AutoReconnect bool
}

type CommandResponse struct {
	Command    string                 `json:"Command"`
	Successful bool                   `json:"Successful"`
	Data       map[string]interface{} `json:"-"`
}

func Open(address, password string, autoReconnect bool) (*Client, error) {
	client := Client{
		address:       address,
		passwordHash:  hash(password),
		State:         GoRCONClientStateIdle,
		AutoReconnect: autoReconnect,
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return &client, nil
}

func (c *Client) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, DefaultTimeout)
	if err != nil {
		return err
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(30 * time.Second)
	if err != nil {
		return err
	}

	c.conn = conn

	if err := c.auth(); err != nil {
		return err
	}

	c.State = GoRCONClientStateConnected

	return nil
}

func (c *Client) Write(command string) (response CommandResponse, err error) {
	if !c.isConnected() {
		if !c.AutoReconnect {
			err = errors.New("connection closed")
			return
		}
		err = c.Connect()
		if err != nil {
			return
		}
	}
	if err = c.write(command); err != nil {
		return
	}
	r, err := c.read()
	if err != nil {
		return
	}
	fmt.Println(r)
	err = json.Unmarshal([]byte(r), &response)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(r), &response.Data)
	delete(response.Data, "Command")
	delete(response.Data, "Successful")
	return
}

func (c *Client) auth() error {
	// server sends "password:" right after connecting
	tmp, err := c.read()
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(tmp), "password") {
		if err := c.write(c.passwordHash); err != nil {
			return err
		}
	} else {
		return errors.New("no password prompt received")
	}

	tmp, err = c.read()
	if err != nil {
		return err
	}

	if !strings.Contains(strings.ToLower(tmp), "authenticated=1") {
		return errors.New("authentication failed")
	}

	return nil
}

func (c *Client) read() (string, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return "", err
	}

	tmp := make([]byte, readLength)
	data := make([]byte, 0)
	for {
		n, err := c.conn.Read(tmp)
		if err != nil {
			return "", err
		}

		data = append(data, tmp[:n]...)

		if n < readLength {
			break // done
		}
	}

	return string(data), nil
}

func (c *Client) write(in string) (err error) {
	_, err = c.conn.Write([]byte(in))
	return err
}

func (c *Client) isConnected() bool {
	// https://stackoverflow.com/questions/12741386/how-to-know-tcp-connection-is-closed-in-net-package
	_ = c.conn.SetReadDeadline(time.Now())
	one := make([]byte, 1)
	if _, err := c.conn.Read(one); err == io.EOF {
		_ = c.conn.Close()
		return false
	}
	return true
}

func hash(in string) string {
	tmp := md5.Sum([]byte(in))
	return hex.EncodeToString(tmp[:])
}
