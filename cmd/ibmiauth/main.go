
package main

import (
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/plugin"
	ibmiauth "github.com/IBM/HashiCorp-IBMi-plugin"
)

func main() {
	// PluginAPIClientMeta is a helper that plugins can use to configure TLS connections back to Vault.
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	// These are set by the command line flags.
	// flagCACert     string
	// flagCAPath     string
	// flagClientCert string
	// flagClientKey  string
	// flagServerName string
	// flagInsecure   bool

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.ServeMultiplex(&plugin.ServeOpts{
		BackendFactoryFunc: ibmiauth.Factory,
		// set the TLSProviderFunc so that the plugin maintains backwards
		// compatibility with Vault versions that don't support plugin AutoMTLS
		TLSProviderFunc: tlsProviderFunc,
	}); err != nil {
		log.Fatal(err)
	}
}
