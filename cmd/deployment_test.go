package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestDeploymentEnv(t *testing.T) {
	for _, source := range []string{"default", "config", "neoshowcase", "explicit"} {
		t.Run(source, func(t *testing.T) {
			v := viper.New()
			v.SetEnvPrefix("TRAQ")
			v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
			v.AutomaticEnv()
			v.SetDefault("mariadb.host", "localhost")
			v.SetDefault("mariadb.port", 3306)
			bindDeploymentEnv(v)
			for _, key := range []string{"TRAQ_MARIADB_HOST", "NS_MARIADB_HOSTNAME", "TRAQ_MARIADB_PORT", "NS_MARIADB_PORT", "TRAQ_MARIADB_USERNAME", "NS_MARIADB_USER", "TRAQ_MARIADB_PASSWORD", "NS_MARIADB_PASSWORD", "TRAQ_MARIADB_DATABASE", "NS_MARIADB_DATABASE", "TRAQ_INITIALADMINPASSWORD"} {
				t.Setenv(key, "")
			}
			want := "localhost"
			if source != "default" {
				v.SetConfigType("yaml")
				require.NoError(t, v.ReadConfig(strings.NewReader("mariadb:\n  host: file-db\n")))
				want = "file-db"
			}
			if source == "neoshowcase" || source == "explicit" {
				t.Setenv("NS_MARIADB_HOSTNAME", "ns-db")
				t.Setenv("NS_MARIADB_PORT", "3307")
				t.Setenv("NS_MARIADB_USER", "ns-user")
				t.Setenv("NS_MARIADB_PASSWORD", "ns-password")
				t.Setenv("NS_MARIADB_DATABASE", "ns-database")
				want = "ns-db"
			}
			if source == "explicit" {
				t.Setenv("TRAQ_MARIADB_HOST", "explicit-db")
				want = "explicit-db"
			}
			t.Setenv("TRAQ_INITIALADMINPASSWORD", "test-bootstrap-password")
			var config Config
			require.NoError(t, v.Unmarshal(&config))
			require.Equal(t, want, config.MariaDB.Host)
			require.Equal(t, "test-bootstrap-password", config.InitialAdminPassword)
			if source == "neoshowcase" || source == "explicit" {
				require.Equal(t, 3307, config.MariaDB.Port)
				require.Equal(t, "ns-user", config.MariaDB.Username)
				require.Equal(t, "ns-password", config.MariaDB.Password)
				require.Equal(t, "ns-database", config.MariaDB.Database)
			}
		})
	}
}

func TestNeoShowcaseBootstrapPassword(t *testing.T) {
	previous := c
	t.Cleanup(func() { c = previous })
	t.Setenv("NS_MARIADB_HOSTNAME", "ns-db")
	cmd := serveCommand()
	for _, password := range []string{"", "traq", "123456789"} {
		c.InitialAdminPassword = password
		require.Error(t, cmd.PreRunE(cmd, nil))
	}
	c.InitialAdminPassword = "random-bootstrap-password"
	require.NoError(t, cmd.PreRunE(cmd, nil))
	t.Setenv("NS_MARIADB_HOSTNAME", "")
	c.InitialAdminPassword = "traq"
	require.NoError(t, cmd.PreRunE(cmd, nil))
}
