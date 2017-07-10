package couchdb

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
)

//Client holds basic information of CouchDB
type Client struct {
	Username string
	Password string
	URL      *url.URL
}

//DB holds information of a database in CouchDB instance
type DB struct {
	client *Client
	Name   string
}

//New returns a new instance of Client
func New(rawurl string, username string, password string) (*Client, error) {
	u, err := url.Parse(rawurl)
	if err == nil {
		client := Client{
			Username: username,
			Password: password,
			URL:      u,
		}
		return &client, nil
	}
	return nil, err
}

//DB returns a database in a given CouchDB instance
func (c *Client) DB(name string) (*DB, error) {
	r, err := url.Parse(name)
	if err == nil {
		db := DB{
			client: c,
			Name:   r.Path,
		}
		return &db, nil
	}
	return nil, err
}

// Sequence represents update sequence ID. It is string in 2.0, integer in previous versions.
// Use a new type to attach a customized unmarshaler
// code borrowed from kivik
type Sequence string

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (id *Sequence) UnmarshalJSON(data []byte) error {
	sid := Sequence(bytes.Trim(data, `""`))
	*id = sid
	return nil
}

// Rev represents CouchDB revision number
type Rev struct {
	Revison string `json:"rev"`
}

// Change represents CouchDB change
type Change struct {
	Seq       Sequence        `json:"seq"`
	ID        string          `json:"id"`
	Revisions []Rev           `json:"changes"`
	Doc       json.RawMessage `json:"doc"`
}

//IChanges is the interface to iterate over changes
type IChanges interface {
	Next() bool
	Get() (*Change, error)
}

//Changes represents the result of CouchDB changes of 'normal' mode
type Changes struct {
	index   int      `json:"-"`
	Results []Change `json:"results"`
	LastReq string   `json:"last_seq"`
	Pending uint     `json:"pending"`
}

//Next returns true when there is data
func (c *Changes) Next() bool {
	c.index++
	return c.index < len(c.Results)
}

//Get return the current data
func (c *Changes) Get() (*Change, error) {
	if c.index < len(c.Results) {
		return &c.Results[c.index], nil
	}
	return nil, errors.New("There is no more change feeds")
}

//ConChanges represents the result stream of a continuous change feeds
type ConChanges struct {
	decoder *json.Decoder
	err     error
	change  Change
}

//Next return true when there is more feeds to come
func (c *ConChanges) Next() bool {
	c.err = c.decoder.Decode(&c.change)
	return c.err == nil
}

//Get return the current feed
func (c *ConChanges) Get() (*Change, error) {
	if c.err == nil {
		return &c.change, nil
	}
	return nil, c.err
}

//ContinuousChanges returns a continous change feeds
func (d *DB) ContinuousChanges(since string) (*ConChanges, error) {
	r, err := url.Parse(d.Name + "/_changes")
	if err == nil {
		u := d.client.URL.ResolveReference(r)
		q := u.Query()
		q.Set("feed", "continuous")
		q.Set("conflicts", "true")
		q.Set("include_docs", "true")
		if len(since) > 0 {
			q.Set("since", since)
		}
		u.RawQuery = q.Encode()
		req, err := http.NewRequest("GET", u.String(), nil)
		//		b, err := httputil.DumpRequestOut(req, true)
		//		pretty.Println(string(b), err)
		if err == nil {
			if len(d.client.Username) > 0 {
				req.SetBasicAuth(d.client.Username, d.client.Password)
			}
			cli := &http.Client{}

			resp, err := cli.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.Status == "200 OK" {
					ch := ConChanges{
						decoder: json.NewDecoder(resp.Body),
					}

					return &ch, nil
				}

				b, _ := httputil.DumpResponse(resp, true)
				return nil, errors.New(string(b[:]))
			}
		}
	}
	return nil, err

}

//NormalChanges returns 100 feeds along with docs and conflicts
func (d *DB) NormalChanges(since string) (*Changes, error) {
	r, err := url.Parse(d.Name + "/_changes")
	if err == nil {
		u := d.client.URL.ResolveReference(r)
		q := u.Query()
		q.Set("feed", "normal")
		q.Set("conflicts", "true")
		q.Set("include_docs", "true")
		q.Set("limit", "100")
		if len(since) > 0 {
			q.Set("since", since)
		}
		u.RawQuery = q.Encode()
		req, err := http.NewRequest("GET", u.String(), nil)
		//		b, err := httputil.DumpRequestOut(req, true)
		//		pretty.Println(string(b), err)
		if err == nil {
			if len(d.client.Username) > 0 {
				req.SetBasicAuth(d.client.Username, d.client.Password)
			}
			cli := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
			resp, err := cli.Do(req)
			if err == nil {
				if resp.Status == "200 OK" {
					data, err := ioutil.ReadAll(resp.Body)
					if err == nil {
						ch := Changes{}
						err = json.Unmarshal(data, &ch)
						if err == nil {
							return &ch, nil
						}
						return nil, err
					}
					return nil, err
				}
				b, _ := httputil.DumpResponse(resp, true)
				return nil, errors.New(string(b[:]))
			}
			return nil, err
		}
		return nil, err
	}
	return nil, err
}
