// +build !plan9

package main

import (
  "os"
  "log"
  "fmt"
  "flag"
  "bytes"
  "strings"
  "net/http"
  "io/ioutil"
  "encoding/base64"
  "github.com/buger/jsonparser"
  "github.com/gorilla/websocket"
)

var debugMode = false;
var version = false;
var containerID = "";
var containerName = "";
var containerLabel = "";
var sendCommand = "";

func main() {
  flag.StringVar(&containerID, "id", "", "ID of container to run command on")
  flag.StringVar(&containerName, "name", "", "Name of container to run command on")
  flag.StringVar(&containerLabel, "label", "", "Label of container group to run command on")
  flag.StringVar(&sendCommand, "cmd", "", "Command to send to container")
  flag.BoolVar(&debugMode, "debug", false, "Enable debug logging")
  flag.BoolVar(&version, "v", false, "Print version")
  flag.Parse()

  if(version){
    fmt.Println("rgon-exec version 1.1.0")
    return
  }

  if isEmpty(sendCommand) { printErr("Missing command") }
  if isEmpty(containerName) && isEmpty(containerID) && isEmpty(containerLabel){
    printErr("Missing container specifier (ID, name, label)")
  }

  // Find container, get execute URL
  executeUrls := []string{}

  if !isEmpty(containerLabel) {
    fmt.Printf("Executing [%s] on label [%s]\n", sendCommand, containerLabel)

    // Multiple containers
    containerExecUrls, _ := getExecUrlsByLabel(containerLabel);
    executeUrls = append(executeUrls, containerExecUrls...)
  }

  // Single containers
  if !isEmpty(containerID) {
    fmt.Printf("Executing [%s] on container [%s]\n", sendCommand, containerID)
    container, _ := getContainerByID(containerID);
    executeUrl, _ := getContainerExecUrl(container);
    executeUrls = append(executeUrls, executeUrl)
  }

  if !isEmpty(containerName) {
    fmt.Printf("Executing [%s] on container [%s]\n", sendCommand, containerName)
    container, _ := getContainerByName(containerName);
    executeUrl, _ := getContainerExecUrl(container);
    executeUrls = append(executeUrls, executeUrl)
  }

  // Process all execute URLs
  for _, url := range executeUrls{
    // Send execute token generation request
    websocketUrl, token, _ := getContainerWsData(url, sendCommand);

    // Run command on websocket
    sendWsExecRequest(websocketUrl, token);
  }

}

func getExecUrlsByLabel(label string) ([]string, error){
  endpoint := "containers?state=running"
  fullData, _ := sendApiRequest(endpoint);
  containers, _, _, _ := jsonparser.Get(fullData, "data")
  urls := []string{}

  jsonparser.ArrayEach(containers, func(container []byte, dataType jsonparser.ValueType, offset int, err error) {
      labels, _, _, _ := jsonparser.Get(container, "labels")

      if(strings.Contains(string(labels), label)){
        exec, _ := jsonparser.GetString(container, "actions", "execute")
        urls = append(urls, exec)
      }
  })

  return urls, nil
}

func getContainerByID(id string) ([]byte, error){
  endpoint := fmt.Sprintf("containers?externalId_prefix=%s", id)
  return sendApiRequest(endpoint)
}

func getContainerByName(name string) ([]byte, error){
  endpoint := fmt.Sprintf("containers?name=%s", name)
  return sendApiRequest(endpoint)
}

func getContainerExecUrl(bodyText []byte) (string, error){
  // Find passed container, get execute URL
  executeUrl, err := jsonparser.GetString(bodyText, "data", "[0]", "actions", "execute")
  if err != nil{
    fmt.Println(err)
    printErr("Couldn't parse container JSON")
    //log.Fatal(err)
  }
  debug("Execute URL: "+ executeUrl)

  return executeUrl, err
}

func getContainerWsData(executeUrl string, sendCommand string) (string, string, error){
  _, key, secret := getCattleVars()

  // Parse command into transmittable format
  cmdFields := strings.Fields(sendCommand)
  cmdParsed := strings.Replace(fmt.Sprintf("%q", cmdFields), " ", ", ", -1)
  cmdStr := fmt.Sprintf(`{ "attachStdin": true, "attachStdout": true, "command": %s, "tty": true }`, cmdParsed)
  debug("Command: "+cmdStr);

  // Send POST with JSON
  req, err := http.NewRequest("POST", executeUrl, bytes.NewBuffer( []byte(cmdStr)))
  req.SetBasicAuth(key, secret)
  req.Header.Set("Content-Type", "application/json")
  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
      panic(err)
  }
  defer resp.Body.Close()
  bodyText, _ := ioutil.ReadAll(resp.Body)

  // Extract the returned Websocket URL
  websocketUrl, err := jsonparser.GetString(bodyText, "url")
  if err != nil{
      log.Fatal(err)
  }

  // Extract the returned token
  token, err := jsonparser.GetString(bodyText, "token")
  if err != nil{
      log.Fatal(err)
  }

  return websocketUrl, token, err;
}

func sendWsExecRequest(websocketUrl string, token string){

  // Add token to provided Websocket URL
  socketTokenUrl := fmt.Sprintf("%s?token=%s", websocketUrl, token)
  debug("socketTokenUrl: "+ socketTokenUrl)

  // Open the websocket
  var dialer *websocket.Dialer
  conn, _, err := dialer.Dial(socketTokenUrl, nil)
  if err != nil {
      log.Fatal(err)
      return
  }

  // Read messages waiting for us in the connection buffer
  // Loops until socket errors or closes
  for {
    _, message, err := conn.ReadMessage()
    if err != nil {
      // Error appears, but this is normal for Rancher
      fmt.Printf("%s\n", err)
      return
    }

    // Decode command exec results from Rancher
    results, err := base64.StdEncoding.DecodeString(string(message))
    if err != nil {
      log.Fatal("Decode error:", err)
      return
    }

    // Print results of command run
    // Only prints if output is available!
    fmt.Printf("%s", results)
  }
}

func sendApiRequest(query string) ([]byte, error){
  url, key, secret := getCattleVars()

  // Build endpoint
  endpoint := fmt.Sprintf("%s/%s", url, query)

  // Get containers from API
  client := &http.Client{}
  req, err := http.NewRequest("GET", endpoint, nil)
  req.SetBasicAuth(key, secret)
  resp, err := client.Do(req)
  if err != nil{
      log.Fatal(err)
  }
  defer resp.Body.Close()
  bodyText, err := ioutil.ReadAll(resp.Body)

  return bodyText, err
}

func getCattleVars() (string, string, string){

  // All variables required=true!
  cattleUrl := getEnvOption("CATTLE_URL", true);
  cattleAccessKey := getEnvOption("CATTLE_ACCESS_KEY", true);
  cattleSecretKey := getEnvOption("CATTLE_SECRET_KEY", true);

  return cattleUrl, cattleAccessKey, cattleSecretKey
}

func getEnvOption(name string, required bool) string {
  val := os.Getenv(name)
  if required && len(val) == 0 {
    log.Fatal("Required environment variable not set: ", name)
  }
  return strings.TrimSpace(val)
}

func isEmpty(str string) (bool){
  return str == "";
}

func debug(message string) {
  if(debugMode){
    fmt.Printf("[EXEC]: %s\n", message)
  }
}

func printErr(message string){
  log.Fatal("[EXEC] Fatal: "+message)
}
