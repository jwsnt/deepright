package emailsvc

import (
	"encoding/json"
	"strings"
)

const responseSchemaRaw = "{\n" +
	"  \"type\": \"object\",\n" +
	"  \"properties\": {\n" +
	"    \"content\": {\n" +
	"      \"type\": \"string\",\n" +
	"      \"description\": \"The content is presented in `HTML` format, and the file paths have been replaced with filenames.\"\n" +
	"    },\n" +
	"    \"artifacts\": {\n" +
	"      \"type\": \"array\",\n" +
	"      \"description\": \"All file paths found in the response.\",\n" +
	"      \"items\": {\n" +
	"        \"type\": \"object\",\n" +
	"        \"properties\": {\n" +
	"          \"path\": {\n" +
	"            \"type\": \"string\",\n" +
	"            \"description\": \"Absolute file path.\"\n" +
	"          },\n" +
	"          \"desc\": {\n" +
	"            \"type\": \"string\",\n" +
	"            \"description\": \"Purpose of this file.\"\n" +
	"          }\n" +
	"        },\n" +
	"        \"required\": [\n" +
	"          \"path\",\n" +
	"          \"desc\"\n" +
	"        ]\n" +
	"      }\n" +
	"    },\n" +
	"    \"why_do_this\": {\n" +
	"      \"type\": \"string\",\n" +
	"      \"description\": \"Brief reasoning for executing this command. Must retain granular data and operational logs for process summarization. The more granular, the better.\"\n" +
	"    }\n" +
	"  },\n" +
	"  \"required\": [\n" +
	"    \"content\",\n" +
	"    \"why_do_this\"\n" +
	"  ]\n" +
	"}"

func SupportedCommands() []string {
	return []string{"command", "help", "name", "param", "scope", "schema", "init", "send", "start", "stop"}
}

func SupportedScopes() []string {
	return []string{"reuse", "agent", "provider", "thinking", "swarm"}
}

func ResponseSchemaJSON() string {
	return strings.TrimSpace(responseSchemaRaw)
}

func ResponseSchemaDefinition() map[string]any {
	var schema map[string]any
	_ = json.Unmarshal([]byte(ResponseSchemaJSON()), &schema)
	return schema
}
