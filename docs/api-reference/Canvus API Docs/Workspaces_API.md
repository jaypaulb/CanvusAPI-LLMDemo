```markdown
## Workspaces API

The workspaces API operates on the workspaces on a client.

### List Workspaces

Gets all workspaces of the specified client.

```bash
GET /clients/:client_id/workspaces
```

| Attribute | Type | Required | Description |
| --- | --- | --- | --- |
| client_id (path) | uuid | yes | ID of the client |
| subscribe (query) | boolean | no | See Streaming |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/clients/e5cad8d4-7051-4051-97bc-13e41fd81ca7/workspaces
```

**Example Response**:
```json
[
  {
    "canvas_id": "df172fda-5e85-4b5c-b602-2eda5a6602f1",
    "canvas_size": {
      "height": 5400,
      "width": 9600
    },
    "index": 0,
    "info_panel_visible": true,
    "location": {
      "x": 0,
      "y": 0
    },
    "pinned": false,
    "server_id": "71ac3376-5020-4033-a159-71c213dc9be6",
    "size": {
      "height": 1080,
      "width": 1271
    },
    "state": "normal",
    "user": "guest",
    "view_rectangle": {
      "height": 648,
      "width": 762.6,
      "x": -2244.5,
      "y": -1080.0
    },
    "workspace_name": "Workspace 1",
    "workspace_state": "open"
  },
  {
    "canvas_id": "",
    "index": 1,
    "info_panel_visible": true,
    "location": {
      "x": 1271,
      "y": 0
    },
    "pinned": false,
    "server_id": "",
    "size": {
      "height": 1080,
      "width": 649
    },
    "state": "normal",
    "user": "",
    "view_rectangle": {
      "height": 648,
      "width": 389.4,
      "x": -3191,
      "y": -1080.0
    },
    "workspace_name": "Workspace 2",
    "workspace_state": "canvas_list"
  }
]
```

### Single Workspace

Gets a single workspace.

```bash
GET /clients/:client_id/workspaces/:workspace_index
```

| Attribute | Type | Required | Description |
| --- | --- | --- | --- |
| client_id (path) | uuid | yes | ID of the client |
| workspace_index (path) | number | yes | Index of the workspace to get |
| subscribe (query) | boolean | no | See Streaming |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/clients/e5cad8d4-7051-4051-97bc-13e41fd81ca7/workspaces/0
```

**Example Response**:
```json
{
  "canvas_id": "df172fda-5e85-4b5c-b602-2eda5a6602f1",
  "canvas_size": {
    "height": 5400,
    "width": 9600
  },
  "index": 0,
  "info_panel_visible": true,
  "location": {
    "x": 0,
    "y": 0
  },
  "pinned": false,
  "server_id": "71ac3376-5020-4033-a159-71c213dc9be6",
  "size": {
    "height": 1080,
    "width": 1271
  },
  "state": "normal",
  "user": "guest",
  "view_rectangle": {
    "height": 648,
    "width": 762.6,
    "x": -2244.5,
    "y": -1080.0
  },
  "workspace_name": "Workspace 1",
  "workspace_state": "open"
}
```

### Update Workspace

Changes some of the workspace parameters such as viewport location. For opening a new canvas use the [open canvas](https://developer.multitaction.com/mt-canvus/workspaces.html#api-workspaces-open) endpoint.

```bash
PATCH /clients/:client_id/workspaces/:workspace_index
```

| Attribute | Type | Required | Description |
| --- | --- | --- | --- |
| client_id (path) | uuid | yes | ID of the client |
| workspace_index (path) | number | yes | Index of the workspace to update |
| info_panel_visible | boolean | no | Is the workspace info panel visible |
| pinned | boolean | no | Is the workspace pinned |
| view_rectangle | object | no | The visible rectangle in canvas coordinates |

**Example cURL Request**:
```bash
curl -X PATCH -H "Private-Token: <access token>" -d '{"pinned":"false"}' https://canvus.example.com/api/v1/clients/e5cad8d4-7051-4051-97bc-13e41fd81ca7/workspaces/0
```

**Example Response**:
```json
{
  "canvas_id": "df172fda-5e85-4b5c-b602-2eda5a6602f1",
  "canvas_size": {
    "height": 5400,
    "width": 9600
  },
  "index": 0,
  "info_panel_visible": true,
  "location": {
    "x": 0,
    "y": 0
  },
  "pinned": false,
  "server_id": "71ac3376-5020-4033-a159-71c213dc9be6",
  "size": {
    "height": 1080,
    "width": 1271
  },
  "state": "normal",
  "user": "guest",
  "view_rectangle": {
    "height": 648,
    "width": 762.6,
    "x": -2244.5,
    "y": -1080.0
  },
  "workspace_name": "Workspace 1",
  "workspace_state": "open"
}
```