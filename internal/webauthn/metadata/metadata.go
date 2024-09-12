package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

const MDS3URL = "https://mds3.fidoalliance.org/"

type MetadataStatement struct {
	CryptoStrength              float64                     `json:"cryptoStrength"`
	AttachmentHint              []string                    `json:"attachmentHint"`
	AttestationRootCertificates []string                    `json:"attestationRootCertificates"`
	Description                 string                      `json:"description"`
	Schema                      float64                     `json:"schema"`
	UserVerificationDetails     [][]UserVerificationDetails `json:"userVerificationDetails"`
	MatcherProtection           []string                    `json:"matcherProtection"`
	AuthenticatorVersion        float64                     `json:"authenticatorVersion"`
	AuthenticationAlgorithms    []string                    `json:"authenticationAlgorithms"`
	TcDisplay                   []string                    `json:"tcDisplay"`
	Icon                        string                      `json:"icon"`
	AuthenticatorGetInfo        AuthenticatorGetInfo        `json:"authenticatorGetInfo"`
	LegalHeader                 string                      `json:"legalHeader"`
	Aaguid                      uuid.UUID                   `json:"aaguid"`
	Upv                         []Upv                       `json:"upv"`
	KeyProtection               []string                    `json:"keyProtection"`
	ProtocolFamily              string                      `json:"protocolFamily"`
	PublicKeyAlgAndEncodings    []string                    `json:"publicKeyAlgAndEncodings"`
	AttestationTypes            []string                    `json:"attestationTypes"`
	TcDisplayContentType        string                      `json:"tcDisplayContentType"`
}

type UserVerificationDetails struct {
	UserVerificationMethod string `json:"userVerificationMethod"`
	CaDesc                 CaDesc `json:"caDesc"`
}

type CaDesc struct {
	MinLength     float64 `json:"minLength"`
	MaxRetries    float64 `json:"maxRetries"`
	BlockSlowdown float64 `json:"blockSlowdown"`
	Base          float64 `json:"base"`
}

type StatusReports struct {
	Status        string `json:"status"`
	EffectiveDate string `json:"effectiveDate"`
}

type Root struct {
	NextUpdate  string    `json:"nextUpdate"`
	Entries     []Entries `json:"entries"`
	LegalHeader string    `json:"legalHeader"`
	No          float64   `json:"no"`
}

type Entries struct {
	Aaguid                 uuid.UUID         `json:"aaguid"`
	MetadataStatement      MetadataStatement `json:"metadataStatement"`
	StatusReports          []StatusReports   `json:"statusReports"`
	TimeOfLastStatusChange string            `json:"timeOfLastStatusChange"`
}

type Upv struct {
	Minor float64 `json:"minor"`
	Major float64 `json:"major"`
}

type AuthenticatorGetInfo struct {
	PinUvAuthProtocols []float64 `json:"pinUvAuthProtocols"`
	Versions           []string  `json:"versions"`
	Extensions         []string  `json:"extensions"`
	Aaguid             uuid.UUID `json:"aaguid"`
	Options            Options   `json:"options"`
	MaxMsgSize         float64   `json:"maxMsgSize"`
}

type Options struct {
	Rk        bool `json:"rk"`
	ClientPin bool `json:"clientPin"`
	Up        bool `json:"up"`
	Uv        bool `json:"uv"`
}

func DownloadFIDO2MetaData(c *http.Client) (*Root, error) {
	var root Root

	resp, err := c.Get(MDS3URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&root)
	if err != nil {
		return nil, err
	}

	return &root, nil
}
