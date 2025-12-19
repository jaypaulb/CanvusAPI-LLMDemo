```markdown
# Connectors API

## List Connectors

Gets all connectors for the specified canvas.

```bash
GET /canvases/:id/connectors
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://example.canvusmultisite.com/api/v1/canvases/canvas_id/connectors
```

**Example Response**:
```json
[
  {
    "dst": {
      "auto_location": false,
      "id": "9d928b20-2886-4c23-b86f-07d5a444ba72",
      "rel_location": {
        "x": 0,
        "y": 0.5
      },
      "tip": "solid-equilateral-triangle"
    },
    "id": "53489e7b-ca23-4aec-a544-0c27f40387c8",
    "line_color": "#e7e7f2ff",
    "line_width": 5,
    "src": {
      "auto_location": false,
      "id": "fc2f6c60-5831-49b5-bc09-2e553c8cd137",
      "rel_location": {
        "x": 1,
        "y": 1
      },
      "tip": "none"
    },
    "state": "normal",
    "type": "curve",
    "widget_type": "Connector"
  }
]
```

## Single Connector

Gets a single connector.

```bash
GET /canvases/:id/connectors/:connector_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `connector_id` (path) | uuid  | yes      | ID of the connector to get    |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://example.canvusmultisite.com/api/v1/canvases/canvas_id/connectors/53489e7b-ca23-4aec-a544-0c27f40387c8
```

**Example Response**:
```json
{
  "dst": {
    "auto_location": false,
    "id": "9d928b20-2886-4c23-b86f-07d5a444ba72",
    "rel_location": {
      "x": 0,
      "y": 0.5
    },
    "tip": "solid-equilateral-triangle"
  },
  "id": "53489e7b-ca23-4aec-a544-0c27f40387c8",
  "line_color": "#e7e7f2ff",
  "line_width": 5,
  "src": {
    "auto_location": false,
    "id": "fc2f6c60-5831-49b5-bc09-2e553c8cd137",
    "rel_location": {
      "x": 1,
      "y": 1
    },
    "tip": "none"
  },
  "state": "normal",
  "type": "curve",
  "widget_type": "Connector"
}
```

## Create Connector

Creates a new connector.

```bash
POST /canvases/:id/connectors
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `dst`               | object  | yes      | Destination widget information |
| `src`               | object  | yes      | Source widget information     |
| `line_color`        | string  | no       | Color of the connector line   |
| `line_width`        | number  | no       | Width of the connector line   |
| `state`             | string  | no       | State of the connector        |
| `type`              | string  | no       | Type of connector (e.g. curve)|
| `widget_type`       | string  | no       | Type of widget                |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" -d '{"dst":{"auto_location":false,"id":"9d928b20-2886-4c23-b86f-07d5a444ba72","rel_location":{"x":0,"y":0.5},"tip":"solid-equilateral-triangle"},"src":{"auto_location":false,"id":"fc2f6c60-5831-49b5-bc09-2e553c8cd137","rel_location":{"x":1,"y":1},"tip":"none"},"line_color":"#e7e7f2ff","line_width":5,"state":"normal","type":"curve","widget_type":"Connector"}' https://example.canvusmultisite.com/api/v1/canvases/canvas_id/connectors
```

**Example Response**:
```json
{
  "dst": {
    "auto_location": false,
    "id": "9d928b20-2886-4c23-b86f-07d5a444ba72",
    "rel_location": {
      "x": 0,
      "y": 0.5
    },
    "tip": "solid-equilateral-triangle"
  },
  "id": "53489e7b-ca23-4aec-a544-0c27f40387c8",
  "line_color": "#e7e7f2ff",
  "line_width": 5,
  "src": {
    "auto_location": false,
    "id": "fc2f6c60-5831-49b5-bc09-2e553c8cd137",
    "rel_location": {
      "x": 1,
      "y": 1
    },
    "tip": "none"
  },
  "state": "normal",
  "type": "curve",
  "widget_type": "Connector"
}
```

## Update Connector

Updates a connector.

```bash
PATCH /canvases/:id/connectors/:connector_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `connector_id` (path) | uuid  | yes      | ID of the connector to update |
| `dst`               | object  | no       | Destination widget information |
| `src`               | object  | no       | Source widget information     |
| `line_color`        | string  | no       | Color of the connector line   |
| `line_width`        | number  | no       | Width of the connector line   |
| `state`             | string  | no       | State of the connector        |
| `type`              | string  | no       | Type of connector (e.g. curve)|
| `widget_type`       | string  | no       | Type of widget                |

**Example cURL Request**:
```bash
curl -X PATCH -H "Private-Token: <access token>" -d '{"line_color":"#ffffff","line_width":7}' https://example.canvusmultisite.com/api/v1/canvases/canvas_id/connectors/53489e7b-ca23-4aec-a544-0c27f40387c8
```

**Example Response**:
```json
{
  "dst": {
    "auto_location": false,
    "id": "9d928b20-2886-4c23-b86f-07d5a444ba72",
    "rel_location": {
      "x": 0,
      "y": 0.5
    },
    "tip": "solid-equilateral-triangle"
  },
  "id": "53489e7b-ca23-4aec-a544-0c27f40387c8",
  "line_color": "#ffffff",
  "line_width": 7,
  "src": {
    "auto_location": false,
    "id": "fc2f6c60-5831-49b5-bc09-2e553c8cd137",
    "rel_location": {
      "x": 1,
      "y": 1
    },
    "tip": "none"
  },
  "state": "normal",
  "type": "curve",
  "widget_type": "Connector"
}
```

## Delete Connector

Deletes a connector.

```bash
DELETE /canvases/:id/connectors/:connector_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `connector_id` (path) | uuid  | yes      | ID of the connector to delete |

**Example cURL Request**:
```bash
curl -X DELETE -H "Private-Token: <access token>" https://example.canvusmultisite.com/api/v1/canvases

/canvas_id/connectors/53489e7b-ca23-4aec-a544-0c27f40387c8
```