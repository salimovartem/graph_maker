package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/corezoid/gitcall-go-runner/gitcall"
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"graph_maker/controlapi"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

import _ "embed" //do not delete

var once sync.Once
var client *openai.Client

// Generate the JSON schema at initialization time
var Schema = GenerateSchema[Graph]()

// A struct that will be converted to a Structured Outputs response schema
type Graph struct {
	Nodes []Node `json:"nodes" jsonschema_description:"The nodes in the graph"`
	Edges []Edge `json:"edges" jsonschema_description:"The edges in the graph"`
}
type Node struct {
	ID   string `json:"id" jsonschema_description:"The unique identifier of the node"`
	Name string `json:"name" jsonschema_description:"The name of the node"`
	X    int    `json:"x" jsonschema_description:"The x-coordinate of the node, tree view of graph, the distance between actors must be no less than 3, 0 - center"`
	Y    int    `json:"y" jsonschema_description:"The y-coordinate of the node, tree view of graph, the distance between actors must be no less than 3, 0 - center"`
}
type Edge struct {
	Source string `json:"source" jsonschema_description:"The source node of the edge"`
	Target string `json:"target" jsonschema_description:"The target node of the edge"`
}

func GenerateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

func main() {
	gitcall.Handle(usercode)
}

type Request struct {
	EventActorID string
	GraphFormID  int
	LayerFormID  int
	LinkType     int
	FormID       int
	Ref          string
	OpenAPIKey   string
	SystemMsg    string
	UserMsg      string
	ChunkSize    int
	Users        []int
	SimAPIKey    string
	WorkspaceID  string
}

func usercode(ctx context.Context, data1 map[string]any) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			stack := StackTrace()
			err := fmt.Errorf("panic: %v %s", r, stack)
			panic(err)

		}
	}()
	fmt.Println("data >", data1)
	if data1["graph_maker_req"] == nil {
		return fmt.Errorf("no graph_maker field")
	}
	gmReq := data1["graph_maker_req"].(map[string]any)
	if gmReq["open_api_key"] == nil {
		return fmt.Errorf("no open_api_key field")
	}
	if gmReq["system_msg"] == nil {
		gmReq["system_msg"] = "You are an expert in creating detailed graphs. You know how to arrange(visualize) actors on a graph beautifully."
	}
	if gmReq["user_msg"] == nil {
		return fmt.Errorf("no msg field")
	}
	if gmReq["chunk_size"] == nil {
		gmReq["chunk_size"] = 20000000.0
	}
	if gmReq["users"] == nil || len(gmReq["users"].([]any)) == 0 {
		return fmt.Errorf("no users field")
	}
	if gmReq["sim_api_key"] == nil {
		return fmt.Errorf("no sim_api_key field")
	}
	if gmReq["workspace_id"] == nil {
		return fmt.Errorf("no workspace_id field")
	}
	if gmReq["ref"] == nil {
		return fmt.Errorf("no ref field")
	}

	req := Request{
		Ref:         gmReq["ref"].(string),
		OpenAPIKey:  gmReq["open_api_key"].(string),
		SystemMsg:   gmReq["system_msg"].(string),
		UserMsg:     gmReq["user_msg"].(string),
		ChunkSize:   int(gmReq["chunk_size"].(float64)),
		SimAPIKey:   gmReq["sim_api_key"].(string),
		WorkspaceID: gmReq["workspace_id"].(string),
	}
	if id, ok := gmReq["event_actor_id"].(string); ok {
		req.EventActorID = id
	}
	users := gmReq["users"].([]any)
	for _, uBin := range users {
		u, err := strconv.Atoi(uBin.(string))
		if err != nil {
			return fmt.Errorf("failed to parse user ID: %v", err)
		}
		req.Users = append(req.Users, u)
	}

	initOnce(req)
	graph := handle(ctx, req)
	graphJSON, err := json.Marshal(graph)
	if err != nil {
		return fmt.Errorf("failed to marshal graph: %v", err)
	}

	// Convert the JSON to map[string]any
	var graphMap map[string]any
	err = json.Unmarshal(graphJSON, &graphMap)
	if err != nil {
		return fmt.Errorf("failed to unmarshal graph JSON: %v", err)
	}
	data1["graph_maker_rsp"] = graphMap

	return nil
}

func initOnce(req Request) {
	once.Do(func() {

		controlapi.Token = req.SimAPIKey
		client = openai.NewClient(option.WithAPIKey(req.OpenAPIKey))
	})
}

func handle(ctx context.Context, req Request) Graph {
	rsp := controlapi.SystemForms(req.WorkspaceID)
	if rsp["data"] == nil {
		panic("no forms")
	}
	forms := rsp["data"].([]any)
	for _, form1 := range forms {
		form := form1.(map[string]any)
		if form["title"].(string) == "Graphs" {
			req.GraphFormID = int(form["id"].(float64))
			continue
		}
		if form["title"].(string) == "Layers" {
			req.LayerFormID = int(form["id"].(float64))
			continue
		}
	}
	rspCustom := controlapi.CustomForms(req.WorkspaceID)
	if rspCustom["data"] == nil {
		panic("no custom forms")
	}
	formsCustom := rspCustom["data"].([]any)
	for _, form1 := range formsCustom {
		form := form1.(map[string]any)
		if form["title"].(string) == "GraphMakerForm" {
			req.FormID = int(form["id"].(float64))
			break
		}
	}
	if req.FormID == 0 {
		req.FormID = controlapi.CreateTemplate(req.WorkspaceID, "GraphMakerForm", []map[string]any{})
	}

	linksType := controlapi.GetTypeLinks(req.WorkspaceID)
	if linksType["data"] == nil {
		panic("no links")
	}
	links := linksType["data"].([]any)
	for _, link1 := range links {
		link := link1.(map[string]any)
		if link["name"].(string) == "hierarchy" {
			req.LinkType = int(link["id"].(float64))
		}
	}

	//rsp1 := controlapi.GetActor(req.WorkspaceID)
	//fmt.Println(rsp1)
	chunks := splitIntoChunks(req.UserMsg, req.ChunkSize)
	gid, lid := prepareGraph(req)
	for _, chunk := range chunks {
		schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:        openai.F("structured_output"),
			Description: openai.F("The structured output of the model"),
			Schema:      openai.F(Schema),
			Strict:      openai.Bool(true),
		}

		// Query the Chat Completions API
		chat, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(req.SystemMsg),
				openai.UserMessage(chunk),
			}),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(schemaParam),
				},
			),
			// Only certain models can perform structured outputs
			Model: openai.F(openai.ChatModelGPT4o2024_08_06),
		})

		if err != nil {
			panic(err.Error())
		}

		// The model responds with a JSON string, so parse it into a struct
		graph := Graph{}
		err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &graph)
		if err != nil {
			panic(err.Error())
		}

		// Use the model's structured response with a native Go struct
		//prettified, err := json.MarshalIndent(graph, "", "  ")
		//if err != nil {
		//	panic(err.Error())
		//}
		//fmt.Println(string(prettified))

		makeGraph(lid, req, graph)
		if req.EventActorID != "" {
			linkToGraph :=
				fmt.Sprintf("https://sim.simulator.company/actors_graph/%s/graph/%s/layers/%s", req.WorkspaceID, gid, lid)
			controlapi.CreateComment(req.EventActorID, "The graph is created based on the event content:\r\n"+linkToGraph)
		}
		return graph

	}
	panic("chunking not implemented")

}

type Info struct {
	laID float64
	id   string
}

var linksRefs = make(map[string]Info)
var linkLLMID = make(map[string]string)

func prepareGraph(req Request) (string, string) {
	gid := controlapi.CreateActor("", req.Ref, req.GraphFormID, map[string]any{}, nil, nil, "")
	for _, userID := range req.Users {
		controlapi.AddAccessString("actor", gid, userID)
	}
	lid := controlapi.CreateLayerActor("Layer", req.LayerFormID)
	controlapi.CreateLink(req.LinkType, req.WorkspaceID, gid, lid)
	for _, userID := range req.Users {
		controlapi.AddAccessString("actor", lid, userID)
	}
	return gid, lid

}

func makeGraph(lid string, req Request, graph Graph) {
	for _, n := range graph.Nodes {
		ref := req.Ref + "." + n.Name
		ref = url.QueryEscape(ref)
		//instanceActor := controlapi.GetActorByRef(req.FormID, ref)
		//
		//if instanceActor["data"] != nil {
		//	linkLLMID[n.ID] = ref
		//	continue
		//}
		id := controlapi.CreateActor(ref, n.Name, req.FormID, map[string]any{}, nil, nil, "")
		laID := controlapi.AddToLayer("node", id, lid, n.X, n.Y)
		linkLLMID[n.ID] = ref
		linksRefs[ref] = Info{laID: laID, id: id}

	}
	for _, e := range graph.Edges {

		id := controlapi.CreateLink(req.LinkType, req.WorkspaceID, getActor(e.Source).id, getActor(e.Target).id)
		fmt.Println(getActor(e.Source), getActor(e.Target), id)
		controlapi.AddToLayer1("edge", id, lid, getActor(e.Source).laID, getActor(e.Target).laID)
	}

}

func getActor(id string) Info {
	rsp := linksRefs[linkLLMID[id]]
	if rsp.id == "" {
		fmt.Println("no actor", id)
		panic("no actor")
	}
	return rsp
}

func splitIntoChunks(s string, size int) []string {
	var chunks []string
	runes := []rune(s)
	for len(runes) > size {
		chunks = append(chunks, string(runes[:size]))
		runes = runes[size:]
	}
	chunks = append(chunks, string(runes))
	return chunks
}

func StackTrace() string {

	var builder strings.Builder
	pc := make([]uintptr, 10) // Увеличьте размер, если необходимо больше глубины стека.
	n := runtime.Callers(3, pc)
	if n == 0 {
		return ""
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		file := frame.File
		f := frame.Function
		fList := strings.Split(frame.Function, ".")
		if len(fList) > 0 {
			f = fList[len(fList)-1]
		}
		fileList := strings.Split(frame.File, "/")
		if len(fileList) > 4 {
			file = strings.Join(fileList[len(fileList)-4:], "/")
		}
		builder.WriteString(fmt.Sprintf("/%s:%d %s(); ", file, frame.Line, f))
		if !more {
			break
		}
	}

	return builder.String()
}
