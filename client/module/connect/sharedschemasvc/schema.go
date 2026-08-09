package sharedschemasvc

import "encoding/json"

// ResponseSchema is a shared schema definition for plugin responses.
// This is used by both feishu and email plugins to normalize response schemas.

const responseSchemaRaw = `{
  "type": "object",
  "properties": {
    "content": {
      "type": "string",
      "description": "The content is presented in plain text (not MARKDOWN), and the file paths have been replaced with filenames."
    },
    "artifacts": {
      "type": "array",
      "description": "All file paths found in the response.",
      "items": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string",
            "description": "Absolute file path."
          },
          "desc": {
            "type": "string",
            "description": "Purpose of this file."
          }
        },
        "required": ["path", "desc"]
      }
    },
    "why_do_this": {
      "type": "string",
      "description": "Brief reasoning for executing this command. Must retain granular data and operational logs for process summarization. The more granular, the better."
    }
  },
  "required": ["content"]
}`

func ResponseSchemaJSON() string {
	return responseSchemaRaw
}

func ResponseSchemaDefinition() map[string]any {
	var schema map[string]any
	_ = json.Unmarshal([]byte(responseSchemaRaw), &schema)
	return schema
}
