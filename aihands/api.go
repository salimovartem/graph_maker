package aihands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"image/color"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

var Token = ""

func init() {
	rand.Seed(time.Now().UnixNano())
}

func DeleteActor(id string) {
	do("https://api.control.events/v/1.0/actors/"+id, "DELETE", nil, true)
}
func CreateComment(actorID, text string) {
	data := map[string]any{
		"description": text,
		"data": map[string]any{
			"rating": 0,
		},
	}
	do("https://api.control.events/v/1.0/reactions/comment/"+actorID, "POST", data, true)
}

func GetActorsByFilter(formID string, key, val string) map[string]any {
	req := "https://api.control.events/v/1.0/actors_filters/" + formID + "?limit=200&offset=0"
	if key != "" {
		req += "&q=" + key + "%3D" + val
	}
	return do(req, "GET", nil, true)

}
func AddAccess(objType string, objID int, userID int) {
	AddAccessString(objType, strconv.Itoa(objID), userID)
}
func AddAccessString(objType string, objID string, userID int) {
	//[{"action":"create","data":{"userId":66320,"privs":{"view":true,"modify":true,"remove":true}}}]
	data := []map[string]any{{
		"action": "create",
		"data": map[string]any{
			"userId": userID,
			"privs":  map[string]any{"view": true, "modify": true, "remove": true},
		},
	},
	}
	do("https://api.control.events/v/1.0/access_rules/"+objType+"/"+objID, "POST", data, true)

}

func CreateAWSTemplate(wid, title string, fields []string) int {
	var f []map[string]string

	for _, field := range fields {
		item := map[string]string{
			"class":    "edit",
			"id":       field,
			"required": "true",
			"title":    field,
		}
		f = append(f, item)
	}

	sections := []map[string]any{
		{
			"content": f,
			"title":   "attributes",
		},
	}
	return CreateTemplate(wid, title, sections)
}

func CreateTemplate(wid, title string, sections []map[string]any) int {
	data := map[string]any{
		"title":       title,
		"description": "",
		"settings":    map[string]any{},
		"sections":    sections,
	}
	rsp := do("https://api.control.events/v/1.0/forms/"+wid+"/true", "POST", data, true)
	rspdata := rsp["data"].(map[string]any)
	id := rspdata["id"].(float64)
	return int(id)
}

func GetActorAccount(actorID string) any {
	return do("https://api.control.events/v/1.0/accounts/"+actorID, "GET", map[string]any{}, true)
}

func CreateActorAccount(nameID, actorID string, currencyID int) string {
	req := map[string]any{
		"accountType": "fact",
		"currencyId":  currencyID,
		//"currencyId":      147563,
		"nameId":          nameID,
		"treeCalculation": true,
		"search":          true,
	}
	rsp := do("https://api.control.events/v/1.0/accounts/"+actorID, "POST", req, true)
	id := rsp["data"].([]interface{})[1].(map[string]any)["id"].(string)
	return id
}

func MakeTransaction(amount float64, actorID string) {
	req := map[string]any{

		"amount":  amount,
		"comment": "",
	}
	do("https://api.control.events/v/1.0/transactions/"+actorID, "POST", req, true)
}

func GetLayerActors(layerID string, log bool) LayerActors {
	var rsp LayerActors
	rspMap := do("https://api.control.events/v/1.0/graph_layers/"+layerID, "GET", map[string]any{}, log)
	err := mapstructure.Decode(rspMap["data"], &rsp)
	if err != nil {
		panic(err)
	}
	return rsp
}

func AddToLayer(typeID, actorID, layerID string, x, y int) float64 {
	req := []map[string]any{
		{
			"action": "create",
			"data": map[string]any{
				"id":   actorID,
				"type": typeID,
				"position": map[string]int{
					"x": x * 50,
					"y": y * 50,
				}},
		},
	}
	rsp := do("https://api.control.events/v/1.0/graph_layers/actors/"+layerID, "POST", req, true)
	fmt.Println(rsp)
	rspdata := rsp["data"].(map[string]any)
	nodesMap := rspdata["nodesMap"].([]any)[0]
	fmt.Println(nodesMap)
	laID := nodesMap.(map[string]any)["laId"].(float64)
	return laID
}

func AddToLayer1(typeID, actorID, layerID string, laIdSource, laIdTarget float64) {
	req := []map[string]any{
		{
			"action": "create",
			"data": map[string]any{
				"id":         actorID,
				"type":       typeID,
				"laIdSource": laIdSource,
				"laIdTarget": laIdTarget,
			},
		},
	}
	do("https://api.control.events/v/1.0/graph_layers/actors/"+layerID, "POST", req, true)
}

func GetWorkspaces() {
	do("https://api.control.events/v/1.0/workspaces", "GET", map[string]any{}, true)
}
func GetTypeLinks(accID string) map[string]any {
	return do("https://api.control.events/v/1.0/edge_types/"+accID, "GET", map[string]any{}, true)
}

func CreateLink(edgeTypeID int, wid, source, target string) string {
	if target == "" {
		return ""
	}
	req := map[string]any{
		"source":     source,
		"target":     target,
		"edgeTypeId": edgeTypeID,

		"curveStyle": "curved",
	}
	rsp := do("https://api.control.events/v/1.0/actors/link/"+wid, "POST", req, true)
	data := rsp["data"].(map[string]any)
	id := data["id"].(string)
	return id

}

func CreateActor(ref, title string, formIDInt int, formData map[string]any, rgba *color.RGBA, pictureObject map[string]any, picture string) string {
	formID := strconv.Itoa(formIDInt)
	if ref == "" {
		ref = strconv.Itoa(int(time.Now().UnixNano()))
	}

	req := map[string]any{
		"ref":         ref,
		"title":       title,
		"description": "",
		"picture":     picture,
		"data":        formData,
	}
	if rgba != nil {
		req["color"] = fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B)
	}

	if pictureObject != nil {
		req["pictureObject"] = pictureObject
	}
	rsp := do("https://api.control.events/v/1.0/actors/actor/"+formID+"/", "POST", req, true)
	data := rsp["data"].(map[string]any)
	id := data["id"].(string)
	return id
}

func CreateLayerActor(title string, formIDInt int) string {
	formID := strconv.Itoa(formIDInt)
	req := map[string]any{
		"title": title,
		"data":  map[string]string{"type": "graph"},
	}

	rsp := do("https://api.control.events/v/1.0/actors/actor/"+formID, "POST", req, true)
	data := rsp["data"].(map[string]any)
	id := data["id"].(string)
	return id
}

func SystemForms(wid string) map[string]any {
	return do("https://api.control.events/v/1.0/forms/templates/system/"+wid+"?formTypes=system", "GET", map[string]any{}, true)
}
func CustomForms(wid string) map[string]any {
	return do("https://api.control.events/v/1.0/forms/templates/"+wid, "GET", map[string]any{}, true)
}

func GetActor(id string) map[string]any {
	return do("https://api.control.events/v/1.0/actors/"+id, "GET", map[string]any{}, true)
}
func GetActorByRef(formIDInt int, ref string) map[string]any {
	formID := strconv.Itoa(formIDInt)
	return do("https://api.control.events/v/1.0/actors/ref/"+formID+"/"+ref, "GET", map[string]any{}, true)
}
func GetLayers(wid string) map[string]any {
	return do("https://api.control.events/v/1.0/graph_layers/list/"+wid, "GET", map[string]any{}, true)
}

func UpdateNameActor(title, ID, formID string) {
	req := map[string]any{
		"title": title,
	}
	do("https://api.control.events/v/1.0/actors/actor/"+formID+"/"+ID, "PUT", req, true)
	return
}

//

func UpdateColorActor(rgba *color.RGBA, title, ID, formID string) {
	req := map[string]any{
		"title": title,
		"color": fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B),
	}
	do("https://api.control.events/v/1.0/actors/actor/"+formID+"/"+ID, "PUT", req, true)
	return
}

func UpdateImageActor(filename, title, ID, formID string) {
	req := map[string]any{
		"title":   title,
		"picture": filename,
	}
	do("https://api.control.events/v/1.0/actors/actor/"+formID+"/"+ID, "PUT", req, true)
	return
}
func CreateImageActor(file io.Reader, wid string, formID int, fileName string, height, width int, layerID string, x, y float64, data map[string]any) string {
	rsp := DownloadFile(wid, file, fileName)
	imj := rsp["data"].(map[string]any)["fileName"]
	pictureObject := map[string]any{
		"height": height,
		"img":    imj,
		"type":   "image",
		"width":  width,
	}
	id := CreateActor("", fileName, formID, data, nil, pictureObject, "")
	AddToLayer("node", id, layerID, int(x), int(y))
	return id
}

func DownloadFile(wid string, file io.Reader, fileName string) map[string]any {
	// {"data":{"title":"upload.png","type":"application/octet-stream","fileName":"tUfDS7lCB","size":15213,"userId":82324,"accId":"149c53c1-43b8-45c6-ad2c-82f25702338d","id":41551}}
	//
	url := "https://api.control.events/v/1.0/upload/" + wid
	//fileDir, _ := os.Getwd()
	//filePath := path.Join(fileDir, fileName)
	//
	//file, _ := os.Open(filePath)
	//defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fileName)
	io.Copy(part, file)
	writer.Close()

	client := &http.Client{}
	r, _ := http.NewRequest("POST", url, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())
	token := "atn_A8fssvtpY8OCLjg1JYKl9olfEDbIjBtq"
	r.Header.Set("Authorization", "Bearer "+token)
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("api req: %s %s \napi rsp: %s\n", url, body.String(), string(bodyBytes))

	var rsp map[string]any
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		panic(err)
	}
	return rsp
}

func do(path, method string, data any, log bool) map[string]any {
	//token := "atn_A8fssvtpY8OCLjg1JYKl9olfEDbIjBtq"
	//if strings.Contains(path, "admin.control.events") {
	//	token = config.GetConfig().Token
	//}
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	client := &http.Client{}
	r := bytes.NewReader(body)
	req, _ := http.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+Token)
	//req.Header.Set("Authorization", "Bearer ")
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	if log && (res.StatusCode != 200 && res.StatusCode != 201) {
		fmt.Printf("api req: %s %s \napi rsp: %s\n", path, string(body), string(bodyBytes))
	}
	fmt.Printf("api req: %s %s \napi rsp: %s\n", path, string(body), string(bodyBytes))

	var rsp map[string]any
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		panic(err)
	}
	return rsp

}
