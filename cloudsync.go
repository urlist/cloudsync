// Copyright 2013 Urlist. All rights reserved.
//
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Author: Andrea Di Persio <andrea@urli.st>

package main

import (
    "log"
    "fmt"
    "os/exec"
    "net/http"
    "strings"
    "flag"
    "os"
    "encoding/json"
)

var (
    config   = Config{}
    configFname = flag.String("config", "cloudsync.json", "Path to configuration file")
)

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

type Config struct {
    Port int

    GSUtilCommand string
    BucketPrefix string
}

type Notification struct {
    Action, Bucket, Name string
}

func (n *Notification) BucketName() string {
    clean := func(x string) string {
        return strings.TrimRight(x, "/")
    }

    if config.BucketPrefix == "" {
        return clean(fmt.Sprintf("gs://%s", n.Bucket))
    }

    bucketName := fmt.Sprintf("gs://%s-%s", config.BucketPrefix, n.Bucket)

    return clean(bucketName)
}

func (n *Notification) CloudPut() error {
    cmd := exec.Command(config.GSUtilCommand, "cp", n.Name, n.BucketName())

    if output, err := cmd.CombinedOutput(); err != nil {
        log.Print(string(output))
        return err
    }

    return nil
}

func (n *Notification) CloudDel() error {
    filename := fmt.Sprintf("%s/%s", strings.TrimRight(n.BucketName(), "/"),
                                     strings.TrimLeft(n.Name, "/"))

    cmd := exec.Command(config.GSUtilCommand, "rm", filename)

    if output, err := cmd.CombinedOutput(); err != nil {
        log.Print(string(output))
        return err
    }

    return nil
}

func (n *Notification) Exec() error {
    var err error

    log.Printf("EXEC: %s %s => %s", n.Action, n.BucketName(), n.Name)

    switch n.Action {
        case "put":
            err = n.CloudPut()

        case "delete":
            err = n.CloudDel()
    }

    return err
}

func NotifyHandler(w http.ResponseWriter, r *http.Request) {
    qs := r.URL.Query()

    notification := Notification{qs["action"][0], qs["bucket"][0], qs["filename"][0]}

    for _, x := range []string{notification.Action, notification.Bucket, notification.Name} {
        if x == "" {
            http.Error(w, "Wrong arguments", 400)
            return
        }
    }

    err := notification.Exec() 

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

    http.HandleFunc("/", NotifyHandler)

    log.Fatal(http.ListenAndServe(serverAddr, nil))
}
