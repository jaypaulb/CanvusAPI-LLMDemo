# Anchors API

The Anchors API operates on anchors within a canvas.

## List Anchors

Gets all anchors of the specified canvas.

```bash
GET /canvases/:id/anchors
```

| Attribute          | Type    | Required | Description             |
|--------------------|---------|----------|-------------------------|
| `id` (path)        | uuid    | yes      | ID of the canvas         |
| `subscribe` (query) | boolean | no       | See Streaming            |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/09348962-32aa-480d-b3d6-cacef4030ac2/anchors
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

## Single Anchor

Gets a single anchor.

```bash
GET /canvases/:id/anchors/:anchor_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `anchor_id` (path)  | uuid    | yes      | ID of the anchor to get       |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/09348962-32aa-480d-b3d6-cacef4030ac2/anchors/5da69125-a5f5-417d-8d8d-a633a0499560
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