// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Shared types are defined here to make structuring better
package azureblobstorage

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// Source, it is the cursor source
type Source struct {
	ContainerName            string
	AccountName              string
	BatchSize                int
	MaxWorkers               int
	Poll                     bool
	PollInterval             time.Duration
	TimeStampEpoch           *int64
	FileSelectors            []fileSelectorConfig
	ReaderConfig             readerConfig
	ExpandEventListFromField string
	PathPrefix               string
}

func (s *Source) Name() string {
	return s.AccountName + "::" + s.ContainerName
}

const (
	sharedKeyType        = "sharedKeyType"
	connectionStringType = "connectionStringType"
	oauth2Type           = "oauth2Type"
	jsonType             = "application/json"
	octetType            = "application/octet-stream"
	ndJsonType           = "application/x-ndjson"
	gzType               = "application/x-gzip"
	csvType              = "text/csv"
	encodingGzip         = "gzip"
)

// currently only shared key & connection string types of credentials are supported
type serviceCredentials struct {
	oauth2Creds        *azidentity.ClientSecretCredential
	sharedKeyCreds     *azblob.SharedKeyCredential
	connectionStrCreds string
	cType              string
}

type blobCredentials struct {
	serviceCreds  *serviceCredentials
	blobName      string
	containerName string
}

var allowedContentTypes = map[string]bool{
	jsonType:   true,
	octetType:  true,
	ndJsonType: true,
	gzType:     true,
	csvType:    true,
}
