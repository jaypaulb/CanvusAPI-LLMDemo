Here is the markdown content for the **Widgets API**:

---

# Widgets API

## List Widgets

Gets all widgets of the specified canvas.

```bash
GET /canvases/:id/widgets
```

| Attribute          | Type    | Required | Description             |
|--------------------|---------|----------|-------------------------|
| `id` (path)        | uuid    | yes      | ID of the canvas         |
| `subscribe` (query) | boolean | no       | See Streaming            |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/e5cad8d4-7051-4051-97bc-13e41fd81ca7/widgets
```

**Example Response**:
```json
[
  {
    "anchor_index": 0,
    "anchor_name": "New anchor",
    "depth": 0,
    "id": "5da69125-a5f5-417d-8d8d-a633a0499560",
    "location": {
      "x": 4676.818359375,
      "y": 2432.80126953125
    },
    "parent_id": "08596c44-389c-45d6-9f32-a9940c54b7d9",
    "pinned": false,
    "scale": 1,
    "size": {
      "height": 800,
      "width": 1200
    },
    "state": "normal",
    "widget_type": "Anchor"
  }
]
```

## Single Widget

Gets a single widget.

```bash
GET /canvases/:id/widgets/:widget_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `widget_id` (path)  | uuid    | yes      | ID of the widget to get       |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/e5cad8d4-7051-4051-97bc-13e41fd81ca7/widgets/5da69125-a5f5-417d-8d8d-a633a0499560
```

**Example Response**:
```json
{
  "anchor_index": 0,
  "anchor_name": "New anchor",
  "depth": 0,
  "id": "5da69125-a5f5-417d-8d8d-a633a0499560",
  "location": {
    "x": 4676.818359375,
    "y": 2432.80126953125
  },
  "parent_id": "08596c44-389c-45d6-9f32-a9940c54b7d9",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 800,
    "width": 1200
  },
  "state": "normal",
  "widget_type": "Anchor"
}
```

## Create Widget

Creates a widget.

```bash
POST /canvases/:id/widgets
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `depth`             | number  | no       | Depth of the widget compared to its siblings |
| `location`          | object  | no       | Location of the widget relative to its parent |
| `pinned`            | boolean | no       | Is the widget pinned or not   |
| `scale`             | number  | no       | Scale of the widget           |
| `size`              | object  | no       | Size of the widget            |
| `widget_type`       | string  | yes      | Type of the widget            |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/e5cad8d4-7051-4051-97bc-13e41fd81ca7/widgets
```

**Example Response**:
```json
{
  "anchor_index": 1,
  "anchor_name": "New anchor",
  "depth": 0,
  "id": "8cf37672-e642-449d-881e-6ee6e42afc16",
  "location": {
    "x": 4632.12841796875,
    "y": 3039.445556640625
  },
  "parent_id": "08596c44-389c-45d6-9f32-a9940c54b7d9",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 800,
    "width": 1200
  },
  "state": "normal",
  "widget_type": "Anchor"
}
```

## Update Widget

Updates a widget.

```bash
PATCH /canvases/:id/widgets/:widget_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `widget_id` (path)  | uuid    | yes      | ID of the widget to update    |
| `depth`             | number  | no       | Depth of the widget compared to its siblings |
| `location`          | object  | no       | Location of the widget relative to its parent |
| `pinned`            | boolean | no       | Is the widget pinned or not   |
| `scale`             | number  | no       | Scale of the widget           |
| `size`              | object  | no       | Size of the widget            |

**Example cURL Request**:
```bash
curl -X PATCH -H "Private-Token: <access token>" -d '{"anchor_name":"Region 1"}' https://canvus.example.com/api/v1/canvases/e5cad8d4-7051-4051-97bc-13e41fd81ca7/widgets/5da69125-a5f5-417d-8d8d-a633a0499560
```

**Example Response**:
```json
{
  "anchor_index": 0,
  "anchor_name": "Region 1",
  "depth": 0,
  "id": "5da69125-a5f5-417d-8d8d-a633a0499560",
  "location": {
    "x": 4676.818359375,
    "y": 2432.80126953125
  },
  "parent_id": "08596c44-389c-45d6-9f32-a9940c54b7d9",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 800,
    "width": 1200
  },
  "state": "normal",
  "widget_type": "Anchor"
}
```

## Delete Widget

Deletes a widget.

```bash
DELETE /canvases/:id/widgets/:widget_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `widget_id` (path)  | uuid    | yes      | ID of the widget to delete    |

**Example cURL Request**:
```bash
curl -X DELETE -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/e5cad8d4-7051-4051-97bc-13e41fd81ca7/widgets/5da69125-a5f5-417d-8d8d-a633a0499560
```