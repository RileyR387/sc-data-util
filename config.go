package main

import (
    "os"
    "strings"
    "fmt"
    "path/filepath"
    log "github.com/sirupsen/logrus"
    "github.com/pborman/getopt/v2"
    "github.com/spf13/viper"
)

var (
    me = filepath.Base(os.Args[0])
    yamlFile = fmt.Sprintf("%s.yaml", me)
    envPrefix = "SCID_UTIL"
    configSearchPaths = []string {".", "./etc", "$HOME/.sc-data-util/", "$HOME/etc", "/etc"}

    stdIn = getopt.BoolLong("stdin", 'i', "Read data from STDIN, Dump to STDOUT. Disables most other options.")
    genConfig = getopt.BoolLong("genconfig", 'x', "Write example config to \"./" + yamlFile + "\"")
    symbol = getopt.StringLong("symbol", 's', "", "Symbol to operate on (required, unless `-i`)")
    startUnixTime = getopt.Int64Long("startUnixTime", 0, 0, "Export Starting at unix time")
    endUnixTime   = getopt.Int64Long("endUnixTime", 0, 0, "End export at unix time")
)

func init() {
    viper.SetConfigName(yamlFile)
    viper.SetConfigType("yaml")
    viper.SetEnvPrefix(envPrefix)
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    viper.AutomaticEnv()

    for _, p := range configSearchPaths {
        viper.AddConfigPath(p)
    }

    viper.SetDefault("data.dir", "data")
    viper.SetDefault("log.level", "TRACE")
    viper.SetDefault("log.file", "")

    getopt.SetUsage(func() { usage() })
    getopt.Parse()

    if *genConfig {
        configWrite()
        os.Exit(0)
        return
    }

    if *stdIn && (*symbol != "" || *startUnixTime != 0 || *endUnixTime != 0) {
        usage( fmt.Sprintf("\nMost options do not work with --stdin input!!\n\n" +
        "Try: %s --genconfig for data directory configuration options.\n", me) )
        os.Exit(1)
    }

    if *symbol == "" && !*stdIn {
        usage( fmt.Sprintf("\nNo input or symbol provided\n\nTry: %s --genconfig\n", me) )
        os.Exit(1)
    }

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            log.Printf("%v\n", err)
            //usage( fmt.Sprintf("\nTry: %s --genconfig\n", me) )
            //os.Exit(1)
        } else {
            log.Fatalf("Failed to parse config: %v\n", err)
            os.Exit(1)
        }
    }

    initLogger( viper.GetString("log.level"), viper.GetString("log.file") )
}

func configWrite(){
    viper.SafeWriteConfigAs( fmt.Sprintf("./%s", yamlFile) )
    log.Printf("Wrote example config to: \"./%s\", feel free to move to: %v", yamlFile, configSearchPaths[1:])
    log.Println(`

Alternatively, set the following environment variables:

export ` + envPrefix + `_DATA_DIR='data'
export ` + envPrefix + `_LOG_LEVEL='TRACE'
export ` + envPrefix + `_LOG_FILE='sc-data-util.log'
`)
    os.Exit(0)
}