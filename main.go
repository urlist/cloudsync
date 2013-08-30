// Copyright 2013 Urlist. All rights reserved.
//
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Author: Andrea Di Persio <andrea@urli.st>

package main

import (
    "os"
    "log"
    "fmt"
    "flag"

    "net/http"

    "encoding/json"

    "github.com/urlist/cloudsync/lib"
)

var (
    config   = Config{}
    configFname = flag.String("config", "cloudsync.json", "Path to configuration file")
)

type Config struct {
    Port int

    GSUtilCommand string
    BucketPrefix string
}

func init() {
    log.SetPrefix("CLOUD ")

    flag.Parse()

    log.Printf("INFO --- Loading configuration from '%v'", *configFname)

    f, err := os.Open(*configFname)

    if err != nil {
        log.Panicf("Cannot read configuration file: %v", err)
    }

    defer f.Close()

    dec := json.NewDecoder(f)

    if err := dec.Decode(&config); err != nil {
        log.Panicf("Cannot decode configuration file: %v", err)
    }
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
    qs := r.URL.Query()

    cloudsync := lib.NewCloudsync(
        config.GSUtilCommand, config.BucketPrefix,
        qs["action"][0], qs["bucket"][0], qs["filename"][0],
    )

    for _, x := range []string{cloudsync.Action, cloudsync.Bucket, cloudsync.Name} {
        if x == "" {
            http.Error(w, "Wrong arguments", 400)
            return
        }
    }

    err := cloudsync.Exec()

    if err != nil {
        log.Printf("FAIL: %s", err)

        http.Error(w, "", 500)
        return
    }

    log.Print("OK")
    fmt.Fprint(w, "OK")
}

func main() {
    log.Print("Listening on port ", config.Port)
    serverAddr := fmt.Sprint(":", config.Port)

    http.HandleFunc("/", rootHandler)

    log.Fatal(http.ListenAndServe(serverAddr, nil))
}
