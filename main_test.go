package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"strconv"
	"testing"
)

//go:embed sample.txt
var Sample string

func TestUserCode_Run(t *testing.T) {
	ctx := context.Background()
	ref := strconv.Itoa(rand.Int())
	dataBin := fmt.Sprintf(`
		{
			"ref": "%s", 
			"graph_maker_req":{
				"event_actor_id": "???",
				"open_api_key": "???", 
				"user_msg": %q, 
				"users": [68381],
				"sim_api_key": "???", 
				"workspace_id": "???"
		    }	
        }
			`,
		ref, Sample)
	data := map[string]any{}
	err := json.Unmarshal([]byte(dataBin), &data)
	require.NoError(t, err)
	err = usercode(ctx, data)
	require.NoError(t, err)
}
