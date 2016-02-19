package iomonkey

import (
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"log"
)

const (
	TOKEN_STORAGE = "/tmp/iomonkey.token"
)

var oauthConfig = &oauth2.Config{
	RedirectURL:  "http://localhost",
	ClientID:     "amzn1.application-oa2-client.808d149075eb494cb8cdb7539209e0dd",
	ClientSecret: "82d005d00a94d585e10a0ac435c3a76ba4340c712e49c7262c01ff3400a364b1",
	Scopes:       []string{"clouddrive:write","clouddrive:read_all"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://www.amazon.com/ap/oa",
		TokenURL: "https://api.amazon.com/auth/o2/token",
	},
}

type fileTokenSource struct {
	path string
	token *oauth2.Token
}

func NewFileTokenSource(path string) *fileTokenSource {
	return &fileTokenSource{
		path: path,
	}
}

func (f *fileTokenSource) Token() (*oauth2.Token, error) {
	if f.token == nil {
		btok, err := ioutil.ReadFile(f.path)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(btok, &f.token)
	}
	return f.token,nil
}

func (f *fileTokenSource) Set(token *oauth2.Token) error {
	f.token = token
	jtok, err := json.Marshal(token)
	if err != nil{
		return err
	}
	return ioutil.WriteFile(f.path, jtok, 0700)
}

func Authorize() (*http.Client, error) {
	fileTokenSrc := NewFileTokenSource(TOKEN_STORAGE)
	tok,err := fileTokenSrc.Token()
	if err != nil {
		log.Println(err)
		// Redirect user to consent page to ask for permission
		// for the scopes specified above.
		url := oauthConfig.AuthCodeURL("")
		fmt.Printf("Visit the URL for the auth dialog: %v\n", url)

		// Use the authorization code that is pushed to the redirect URL.
		// NewTransportWithCode will do the handshake to retrieve
		// an access token and initiate a Transport that is
		// authorized and authenticated by the retrieved token.
		fmt.Printf("Enter code:")
		var code string
		if _, err := fmt.Scan(&code); err != nil {
			return nil, err
		}

		tok, err := oauthConfig.Exchange(oauth2.NoContext, code)
		if err != nil {
			return nil, err
		}

		if fileTokenSrc.Set(tok); err != nil {
			return nil,err
		}
	}

	tok, err = fileTokenSrc.Token()
	if err != nil {
		return nil,err
	}

	tokenSrc := oauthConfig.TokenSource(oauth2.NoContext,tok)
	tok, err = tokenSrc.Token()
	if err != nil {
		return nil,err
	}
	return oauthConfig.Client(oauth2.NoContext, tok), nil

}
