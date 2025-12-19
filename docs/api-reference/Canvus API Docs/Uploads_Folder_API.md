Here is the markdown content for the **Uploads Folder API**:

---

# Uploads Folder API

## Upload a Note

Creates a new note inside the uploads folder. The request must be a multipart POST request with a single `json` part.

```bash
POST /canvases/:id/uploads-folder
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `upload_type`       | string  | yes      | Must be equal to `note`       |
| `background_color`  | string  | no       | Color of the note             |
| `text`              | string  | no       | Text in the note              |
| `title`             | string  | no       | Title of the note             |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" -F 'json={"upload_type":"note","text":"Hello, world","title":"Hello, world"}' https://canvus.example.com/api/v1/canvases/863e7a77-0874-4170-9a0c-7f65d7869c37/uploads-folder
```

**Example Response**:
```json
{
  "background_color": "#ffffff",
  "depth": 0,
  "id": "content",
  "location": {
    "x": 0,
    "y": 0
  },
  "parent_id": "99a81a8b-9605-4122-a233-8de3a03a3f05",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 300,
    "width": 300
  },
  "state": "normal",
  "text": "Hello, world",
  "title": "Hello, world",
  "widget_type": "Note"
}
```

## Upload a File

Creates a new file asset inside the uploads folder. You can upload PDFs, images, videos, and other files. The asset type is automatically recognized by Canvus and you don't need to specify it through the API. Unrecognized file types are uploaded as generic assets.

The request must be a multipart POST request with an optional `json` and mandatory `data` part.

```bash
POST /canvases/:id/uploads-folder
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `upload_type`       | string  | no       | Must be missing, empty, or equal to `asset` |
| `original_filename` | string  | no       | Original filename             |
| `title`             | string  | no       | Title of the asset            |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" -F 'json={"upload_type":"asset","title":"My video"}' -F 'data=@SampleVideo_1280x720_1mb.mp4' https://canvus.example.com/api/v1/canvases/863e7a77-0874-4170-9a0c-7f65d7869c37/uploads-folder
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "",
  "id": "content",
  "location": {
    "x": 0,
    "y": 0
  },
  "original_filename": "",
  "parent_id": "b491c355-1bcc-48ff-be30-6df65d6e1ed4",
  "pinned": false,
  "playback_position": 0,
  "playback_state": "STOPPED",
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "My video",
  "widget_type": "Video"
}
```