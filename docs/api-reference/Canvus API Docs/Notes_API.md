# Notes API

## List Notes

Gets all notes of the specified canvas.

```bash
GET /canvases/:id/notes
```

| Attribute          | Type    | Required | Description             |
|--------------------|---------|----------|-------------------------|
| `id` (path)        | uuid    | yes      | ID of the canvas         |
| `subscribe` (query) | boolean | no       | See Streaming            |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/78cfbcc8-aed9-4bbb-95ca-a0b9a5358d5a/notes
```

**Example Response**:
```json
[
  {
    "background_color": "#ffffff",
    "depth": 0,
    "id": "f77e17a2-3716-46e4-8143-97dde114b20b",
    "location": {
      "x": 3892.455810546875,
      "y": 3326.92041015625
    },
    "parent_id": "fcc7bd6e-3334-4de6-ad95-2c0ab7d47b70",
    "pinned": false,
    "scale": 1,
    "size": {
      "height": 300,
      "width": 300
    },
    "state": "normal",
    "text": "",
    "title": "",
    "widget_type": "Note"
  }
]
```

## Single Note

Gets a single note.

```bash
GET /canvases/:id/notes/:note_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `note_id` (path)    | uuid    | yes      | ID of the note to get        |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/78cfbcc8-aed9-4bbb-95ca-a0b9a5358d5a/notes/f77e17a2-3716-46e4-8143-97dde114b20b
```

**Example Response**:
```json
{
  "background_color": "#ffffff",
  "depth": 0,
  "id": "f77e17a2-3716-46e4-8143-97dde114b20b",
  "location": {
    "x": 3892.455810546875,
    "y": 3326.92041015625
  },
  "parent_id": "fcc7bd6e-3334-4de6-ad95-2c0ab7d47b70",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 300,
    "width": 300
  },
  "state": "normal",
  "text": "",
  "title": "",
  "widget_type": "Note"
}
```

## Create Note

Creates a note.

```bash
POST /canvases/:id/notes
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `background_color`  | string  | no       | Color of the note             |
| `depth`             | number  | no       | Depth of the note compared to its siblings |
| `location`          | object  | no       | Location of the note relative to its parent |
| `parent_id`         | uuid    | no       | ID of the note's parent       |
| `pinned`            | boolean | no       | Is the note pinned or not     |
| `scale`             | number  | no       | Scale of the note             |
| `size`              | object  | no       | Size of the note              |
| `text`              | string  | no       | Text in the note              |
| `title`             | string  | no       | Title of the note             |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/78cfbcc8-aed9-4bbb-95ca-a0b9a5358d5a/notes
```

**Example Response**:
```json
{
  "background_color": "#ffffff",
  "depth": 0,
  "id": "65110c5c-3dd1-4d71-963e-ae1549fa2f9b",
  "location": {
    "x": 5626.251953125,
    "y": 3265.86328125
  },
  "parent_id": "fcc7bd6e-3334-4de6-ad95-2c0ab7d47b70",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 300,
    "width": 300
  },
  "state": "normal",
  "text": "I am note",
  "title": "",
  "widget_type": "Note"
}
```

## Update Note

Updates a note.

```bash
PATCH /canvases/:id/notes/:note_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `note_id` (path)    | uuid    | yes      | ID of the note to update      |
| `background_color`  | string  | no       | Color of the note             |
| `depth`             | number  | no       | Depth of the note compared to its siblings |
| `location`          | object  | no       | Location of the note relative to its parent |
| `parent_id`         | uuid    | no       | ID of the note's parent       |
| `pinned`            | boolean | no       | Is the note pinned or not     |
| `scale`             | number  | no       | Scale of the note             |
| `size`              | object  | no       | Size of the note              |
| `text`              | string  | no       | Text in the note              |
| `title`             | string  | no       | Title of the note             |

**Example cURL Request**:
```bash
curl -X PATCH -H "Private-Token: <access token>" -d '{"text":"I am note"}' https://canvus.example.com/api/v1/canvases/78cfbcc8-aed9-4bbb-95ca-a0b9a5358d5a/notes/f77e17a2-3716-46e4-8143-97dde114b20b
```

**Example Response**:
```json
{
  "background_color": "#ffffff",
  "depth": 0,
  "id": "f77e17a2-3716-46e4-8143-97dde114b20b",
  "location": {
    "x": 3892.455810546875,
    "y": 3326.92041015625
  },
  "parent_id": "fcc7bd6e-3334-4de6-ad95-2c0ab7d47b70",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 300,
    "width": 300
  },
  "state": "normal",
  "text": "I am note",
  "title": "",
  "widget_type": "Note"
}
```

## Delete Note

Deletes a note.

```bash
DELETE /canvases/:id/notes/:note_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `note_id` (path)    | uuid    | yes      | ID of the note to delete      |

**Example cURL Request**:
```bash
curl -X DELETE -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/78cfbcc8-aed9-4bbb-95ca-a0b9a5358d5a/notes/f77e17a2-3716-46e4-8143-97dde114b20b
```

# Notes API Updates

## Note Properties

| Attribute           | Type    | Required | Description                  |
|--------------------|---------|----------|------------------------------|
| `auto_text_color`  | boolean | no       | If true, text color is automatically set based on background color. Default: true |
| `background_color` | string  | no       | Background color (only settable when auto_text_color is false) |
| `text_color`       | string  | no       | Text color (only settable when auto_text_color is false) |

**Note**: When `auto_text_color` is true, attempts to set `background_color` or `text_color` will result in a 409 Conflict error.

# File-based Widgets (PDFs, Images, Videos)

## Common Properties

| Attribute | Type   | Required | Description                  |
|-----------|--------|----------|------------------------------|
| `hash`    | string | no       | System-generated file hash. Read-only, do not include in POST/PATCH requests |