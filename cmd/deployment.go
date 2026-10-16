package cmd

import "github.com/spf13/viper"

func bindDeploymentEnv(v *viper.Viper) {
	// Explicit traQ settings take precedence over NeoShowcase's injected credentials.
	for _, binding := range [][3]string{
		{"mariadb.host", "TRAQ_MARIADB_HOST", "NS_MARIADB_HOSTNAME"},
		{"mariadb.port", "TRAQ_MARIADB_PORT", "NS_MARIADB_PORT"},
		{"mariadb.username", "TRAQ_MARIADB_USERNAME", "NS_MARIADB_USER"},
		{"mariadb.password", "TRAQ_MARIADB_PASSWORD", "NS_MARIADB_PASSWORD"},
		{"mariadb.database", "TRAQ_MARIADB_DATABASE", "NS_MARIADB_DATABASE"},
	} {
		if err := v.BindEnv(binding[0], binding[1], binding[2]); err != nil {
			panic(err)
		}
	}
	v.SetDefault("initialAdminPassword", "traq")
}
