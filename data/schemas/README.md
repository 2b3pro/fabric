# Perplexity

Perplexity provides citation output, so if you want citations in the schema, you'll need to add:

```json
"citations": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {
            "type": "string"
          },
          "url": {
            "type": "string"
          }
        },
        "required": [
          "title",
          "url"
        ]
      }
    }
```