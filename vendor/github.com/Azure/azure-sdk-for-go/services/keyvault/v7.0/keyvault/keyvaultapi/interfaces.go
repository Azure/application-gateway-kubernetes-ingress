package keyvaultapi

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

// BaseClientAPI contains the set of methods on the BaseClient type.
type BaseClientAPI interface {
	BackupCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.BackupCertificateResult, err error)
	BackupKey(ctx context.Context, vaultBaseURL string, keyName string) (result keyvault.BackupKeyResult, err error)
	BackupSecret(ctx context.Context, vaultBaseURL string, secretName string) (result keyvault.BackupSecretResult, err error)
	BackupStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string) (result keyvault.BackupStorageResult, err error)
	CreateCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateCreateParameters) (result keyvault.CertificateOperation, err error)
	CreateKey(ctx context.Context, vaultBaseURL string, keyName string, parameters keyvault.KeyCreateParameters) (result keyvault.KeyBundle, err error)
	Decrypt(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string, parameters keyvault.KeyOperationsParameters) (result keyvault.KeyOperationResult, err error)
	DeleteCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.DeletedCertificateBundle, err error)
	DeleteCertificateContacts(ctx context.Context, vaultBaseURL string) (result keyvault.Contacts, err error)
	DeleteCertificateIssuer(ctx context.Context, vaultBaseURL string, issuerName string) (result keyvault.IssuerBundle, err error)
	DeleteCertificateOperation(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.CertificateOperation, err error)
	DeleteKey(ctx context.Context, vaultBaseURL string, keyName string) (result keyvault.DeletedKeyBundle, err error)
	DeleteSasDefinition(ctx context.Context, vaultBaseURL string, storageAccountName string, sasDefinitionName string) (result keyvault.DeletedSasDefinitionBundle, err error)
	DeleteSecret(ctx context.Context, vaultBaseURL string, secretName string) (result keyvault.DeletedSecretBundle, err error)
	DeleteStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string) (result keyvault.DeletedStorageBundle, err error)
	Encrypt(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string, parameters keyvault.KeyOperationsParameters) (result keyvault.KeyOperationResult, err error)
	GetCertificate(ctx context.Context, vaultBaseURL string, certificateName string, certificateVersion string) (result keyvault.CertificateBundle, err error)
	GetCertificateContacts(ctx context.Context, vaultBaseURL string) (result keyvault.Contacts, err error)
	GetCertificateIssuer(ctx context.Context, vaultBaseURL string, issuerName string) (result keyvault.IssuerBundle, err error)
	GetCertificateIssuers(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.CertificateIssuerListResultPage, err error)
	GetCertificateOperation(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.CertificateOperation, err error)
	GetCertificatePolicy(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.CertificatePolicy, err error)
	GetCertificates(ctx context.Context, vaultBaseURL string, maxresults *int32, includePending *bool) (result keyvault.CertificateListResultPage, err error)
	GetCertificateVersions(ctx context.Context, vaultBaseURL string, certificateName string, maxresults *int32) (result keyvault.CertificateListResultPage, err error)
	GetDeletedCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.DeletedCertificateBundle, err error)
	GetDeletedCertificates(ctx context.Context, vaultBaseURL string, maxresults *int32, includePending *bool) (result keyvault.DeletedCertificateListResultPage, err error)
	GetDeletedKey(ctx context.Context, vaultBaseURL string, keyName string) (result keyvault.DeletedKeyBundle, err error)
	GetDeletedKeys(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.DeletedKeyListResultPage, err error)
	GetDeletedSasDefinition(ctx context.Context, vaultBaseURL string, storageAccountName string, sasDefinitionName string) (result keyvault.DeletedSasDefinitionBundle, err error)
	GetDeletedSasDefinitions(ctx context.Context, vaultBaseURL string, storageAccountName string, maxresults *int32) (result keyvault.DeletedSasDefinitionListResultPage, err error)
	GetDeletedSecret(ctx context.Context, vaultBaseURL string, secretName string) (result keyvault.DeletedSecretBundle, err error)
	GetDeletedSecrets(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.DeletedSecretListResultPage, err error)
	GetDeletedStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string) (result keyvault.DeletedStorageBundle, err error)
	GetDeletedStorageAccounts(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.DeletedStorageListResultPage, err error)
	GetKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string) (result keyvault.KeyBundle, err error)
	GetKeys(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.KeyListResultPage, err error)
	GetKeyVersions(ctx context.Context, vaultBaseURL string, keyName string, maxresults *int32) (result keyvault.KeyListResultPage, err error)
	GetSasDefinition(ctx context.Context, vaultBaseURL string, storageAccountName string, sasDefinitionName string) (result keyvault.SasDefinitionBundle, err error)
	GetSasDefinitions(ctx context.Context, vaultBaseURL string, storageAccountName string, maxresults *int32) (result keyvault.SasDefinitionListResultPage, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	GetSecrets(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultPage, err error)
	GetSecretVersions(ctx context.Context, vaultBaseURL string, secretName string, maxresults *int32) (result keyvault.SecretListResultPage, err error)
	GetStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string) (result keyvault.StorageBundle, err error)
	GetStorageAccounts(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.StorageListResultPage, err error)
	ImportCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error)
	ImportKey(ctx context.Context, vaultBaseURL string, keyName string, parameters keyvault.KeyImportParameters) (result keyvault.KeyBundle, err error)
	MergeCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateMergeParameters) (result keyvault.CertificateBundle, err error)
	PurgeDeletedCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result autorest.Response, err error)
	PurgeDeletedKey(ctx context.Context, vaultBaseURL string, keyName string) (result autorest.Response, err error)
	PurgeDeletedSecret(ctx context.Context, vaultBaseURL string, secretName string) (result autorest.Response, err error)
	PurgeDeletedStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string) (result autorest.Response, err error)
	RecoverDeletedCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.CertificateBundle, err error)
	RecoverDeletedKey(ctx context.Context, vaultBaseURL string, keyName string) (result keyvault.KeyBundle, err error)
	RecoverDeletedSasDefinition(ctx context.Context, vaultBaseURL string, storageAccountName string, sasDefinitionName string) (result keyvault.SasDefinitionBundle, err error)
	RecoverDeletedSecret(ctx context.Context, vaultBaseURL string, secretName string) (result keyvault.SecretBundle, err error)
	RecoverDeletedStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string) (result keyvault.StorageBundle, err error)
	RegenerateStorageAccountKey(ctx context.Context, vaultBaseURL string, storageAccountName string, parameters keyvault.StorageAccountRegenerteKeyParameters) (result keyvault.StorageBundle, err error)
	RestoreCertificate(ctx context.Context, vaultBaseURL string, parameters keyvault.CertificateRestoreParameters) (result keyvault.CertificateBundle, err error)
	RestoreKey(ctx context.Context, vaultBaseURL string, parameters keyvault.KeyRestoreParameters) (result keyvault.KeyBundle, err error)
	RestoreSecret(ctx context.Context, vaultBaseURL string, parameters keyvault.SecretRestoreParameters) (result keyvault.SecretBundle, err error)
	RestoreStorageAccount(ctx context.Context, vaultBaseURL string, parameters keyvault.StorageRestoreParameters) (result keyvault.StorageBundle, err error)
	SetCertificateContacts(ctx context.Context, vaultBaseURL string, contacts keyvault.Contacts) (result keyvault.Contacts, err error)
	SetCertificateIssuer(ctx context.Context, vaultBaseURL string, issuerName string, parameter keyvault.CertificateIssuerSetParameters) (result keyvault.IssuerBundle, err error)
	SetSasDefinition(ctx context.Context, vaultBaseURL string, storageAccountName string, sasDefinitionName string, parameters keyvault.SasDefinitionCreateParameters) (result keyvault.SasDefinitionBundle, err error)
	SetSecret(ctx context.Context, vaultBaseURL string, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error)
	SetStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string, parameters keyvault.StorageAccountCreateParameters) (result keyvault.StorageBundle, err error)
	Sign(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string, parameters keyvault.KeySignParameters) (result keyvault.KeyOperationResult, err error)
	UnwrapKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string, parameters keyvault.KeyOperationsParameters) (result keyvault.KeyOperationResult, err error)
	UpdateCertificate(ctx context.Context, vaultBaseURL string, certificateName string, certificateVersion string, parameters keyvault.CertificateUpdateParameters) (result keyvault.CertificateBundle, err error)
	UpdateCertificateIssuer(ctx context.Context, vaultBaseURL string, issuerName string, parameter keyvault.CertificateIssuerUpdateParameters) (result keyvault.IssuerBundle, err error)
	UpdateCertificateOperation(ctx context.Context, vaultBaseURL string, certificateName string, certificateOperation keyvault.CertificateOperationUpdateParameter) (result keyvault.CertificateOperation, err error)
	UpdateCertificatePolicy(ctx context.Context, vaultBaseURL string, certificateName string, certificatePolicy keyvault.CertificatePolicy) (result keyvault.CertificatePolicy, err error)
	UpdateKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string, parameters keyvault.KeyUpdateParameters) (result keyvault.KeyBundle, err error)
	UpdateSasDefinition(ctx context.Context, vaultBaseURL string, storageAccountName string, sasDefinitionName string, parameters keyvault.SasDefinitionUpdateParameters) (result keyvault.SasDefinitionBundle, err error)
	UpdateSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string, parameters keyvault.SecretUpdateParameters) (result keyvault.SecretBundle, err error)
	UpdateStorageAccount(ctx context.Context, vaultBaseURL string, storageAccountName string, parameters keyvault.StorageAccountUpdateParameters) (result keyvault.StorageBundle, err error)
	Verify(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string, parameters keyvault.KeyVerifyParameters) (result keyvault.KeyVerifyResult, err error)
	WrapKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string, parameters keyvault.KeyOperationsParameters) (result keyvault.KeyOperationResult, err error)
}

var _ BaseClientAPI = (*keyvault.BaseClient)(nil)
