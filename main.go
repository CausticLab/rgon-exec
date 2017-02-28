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
var containerName = "";
var sendCommand = "";

func main() {
  flag.StringVar(&containerName, "name", "", "Name of container to run command on")
  flag.StringVar(&sendCommand, "cmd", "", "Command to send to container")
  flag.BoolVar(&debugMode, "debug", false, "Enable debug logging")
  flag.Parse()

  if(containerName == "") { printErr("Missing container name") }
  if(sendCommand == "") { printErr("Missing command") }

  fmt.Printf("Executing [%s] on container [%s]\n", sendCommand, containerName)
  debug("Finding container: " + containerName)

  // Find container, get execute URL
  executeUrl, _ := getContainerExecUrl(containerName);

  // Send execute token generation request
  websocketUrl, token, _ := getContainerWsData(executeUrl, sendCommand);

  // Run command on websocket
  sendWsExecRequest(websocketUrl, token);

}

func getContainerExecUrl(name string) (string, error){
  url, key, secret := getCattleVars()

  // Get containers from API
  endpoint := fmt.Sprintf("%s/containers?name=%s", url, containerName)
  client := &http.Client{}
  req, err := http.NewRequest("GET", endpoint, nil)
  req.SetBasicAuth(key, secret)
  resp, err := client.Do(req)
  if err != nil{
      log.Fatal(err)
  }
  defer resp.Body.Close()
  bodyText, err := ioutil.ReadAll(resp.Body)

  // Find passed container, get execute URL
  executeUrl, err := jsonparser.GetString(bodyText, "data", "[0]", "actions", "execute")
  if err != nil{
      log.Fatal(err)
  }
  debug("Execute URL: "+ executeUrl)

  return executeUrl, err;
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

func getCattleVars() (string, string, string){

  // All variables required=true!
  cattleUrl := getEnvOption("CATTLE_URL", true);
  cattleAccessKey := getEnvOption("CATTLE_ACCESS_KEY", true);
  cattleSecretKey := getEnvOption("CATTLE_SECRET_KEY", true);

  debug(cattleUrl)

  return cattleUrl, cattleAccessKey, cattleSecretKey
}

func getEnvOption(name string, required bool) string {
  val := os.Getenv(name)
  if required && len(val) == 0 {
    log.Fatal("Required environment variable not set: ", name)
  }
  return strings.TrimSpace(val)
}

func debug(message string) {
  if(debugMode){
    fmt.Printf("[EXEC]: %s\n", message)
  }
}

func printErr(message string){
  log.Fatal("[EXEC] Fatal: "+message)
}
