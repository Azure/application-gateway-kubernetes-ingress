// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type certName string
type certsByName map[certName]n.ApplicationGatewaySslCertificate

// GetBlacklistedCertificates filters the given list of certificates to the list of certs that AGIC is allowed to manage.
// A certificate is considered blacklisted when it is associated with a blacklisted listener.
func (er ExistingResources) GetBlacklistedCertificates() ([]n.ApplicationGatewaySslCertificate, []n.ApplicationGatewaySslCertificate) {
	blacklistedCertSet := er.getBlacklistedCertSet()
	var nonBlacklisted []n.ApplicationGatewaySslCertificate
	var blacklisted []n.ApplicationGatewaySslCertificate
	for _, cert := range er.Certificates {
		// Is the certificate associated with a blacklisted listener?
		if _, exists := blacklistedCertSet[certName(*cert.Name)]; exists {
			glog.V(5).Infof("[brownfield] Certificate %s is blacklisted", certName(*cert.Name))
			blacklisted = append(blacklisted, cert)
			continue
		}
		glog.V(5).Infof("[brownfield] Certificate %s is not blacklisted", certName(*cert.Name))
		nonBlacklisted = append(nonBlacklisted, cert)
	}
	return blacklisted, nonBlacklisted
}

// MergeCerts merges list of lists of certs into a single list, maintaining uniqueness.
func MergeCerts(certBuckets ...[]n.ApplicationGatewaySslCertificate) []n.ApplicationGatewaySslCertificate {
	uniq := make(certsByName)
	for _, bucket := range certBuckets {
		for _, cert := range bucket {
			uniq[certName(*cert.Name)] = cert
		}
	}
	var merged []n.ApplicationGatewaySslCertificate
	for _, cert := range uniq {
		merged = append(merged, cert)
	}
	return merged
}

// LogCertificates emits a few log lines detailing what certificates are created, blacklisted, and removed from ARM.
func LogCertificates(existingBlacklisted []n.ApplicationGatewaySslCertificate, existingNonBlacklisted []n.ApplicationGatewaySslCertificate, managedCertificates []n.ApplicationGatewaySslCertificate) {
	var garbage []n.ApplicationGatewaySslCertificate

	blacklistedSet := indexCertificatesByName(existingBlacklisted)
	managedSet := indexCertificatesByName(managedCertificates)

	for CertificateName, Certificate := range indexCertificatesByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[CertificateName]
		_, existsInNewCertificates := managedSet[CertificateName]
		if !existsInBlacklist && !existsInNewCertificates {
			garbage = append(garbage, Certificate)
		}
	}

	glog.V(3).Info("[brownfield] Certificates AGIC created: ", getCertificateNames(managedCertificates))
	glog.V(3).Info("[brownfield] Existing Blacklisted Certificates AGIC will retain: ", getCertificateNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing Certificates AGIC will remove: ", getCertificateNames(garbage))
}

func getCertificateNames(certificates []n.ApplicationGatewaySslCertificate) string {
	var names []string
	for _, certificate := range certificates {
		names = append(names, *certificate.Name)
	}
	if len(names) == 0 {
		return "n/a"
	}
	return strings.Join(names, ", ")
}

func indexCertificatesByName(certificates []n.ApplicationGatewaySslCertificate) certsByName {
	indexed := make(certsByName)
	for _, cert := range certificates {
		indexed[certName(*cert.Name)] = cert
	}
	return indexed
}

func (er ExistingResources) getBlacklistedCertSet() map[certName]interface{} {
	// Get the list of blacklisted listeners, from which we can determine what certificates should be blacklisted.
	existingBlacklistedListeners, _ := er.GetBlacklistedListeners()
	blacklistedCertSet := make(map[certName]interface{})
	for _, listener := range existingBlacklistedListeners {
		if listener.SslCertificate != nil && listener.SslCertificate.ID != nil {
			certName := certName(utils.GetLastChunkOfSlashed(*listener.SslCertificate.ID))
			blacklistedCertSet[certName] = nil
		}
	}
	return blacklistedCertSet
}
