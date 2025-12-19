# Videos API

## List Videos

Gets all videos of the specified canvas.

```bash
GET /canvases/:id/videos
```

| Attribute          | Type    | Required | Description             |
|--------------------|---------|----------|-------------------------|
| `id` (path)        | uuid    | yes      | ID of the canvas         |
| `subscribe` (query) | boolean | no       | See Streaming            |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a8b5ed9e-6e57-41c8-96cb-be4aec4bfd03/videos
```

**Example Response**:
```json
[
  {
    "depth": 0,
    "hash": "863b0937ee95",
    "id": "105a8486-ac84-49f9-a487-763279a6f658",
    "location": {
      "x": 3737.212646484375,
      "y": 2311.91357421875
    },
    "original_filename": "",
    "parent_id": "34f8909a-6736-49f9-9027-7f7cab5c1acf",
    "pinned": false,
    "playback_position": 0,
    "playback_state": "STOPPED",
    "scale": 1,
    "size": {
      "height": 100,
      "width": 100
    },
    "state": "normal",
    "title": "A video",
    "widget_type": "Video"
  }
]
```

## Single Video

Gets a single video.

```bash
GET /canvases/:id/videos/:video_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `video_id` (path)   | uuid    | yes      | ID of the video to get        |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a8b5ed9e-6e57-41c8-96cb-be4aec4bfd03/videos/105a8486-ac84-49f9-a487-763279a6f658
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "863b0937ee95",
  "id": "105a8486-ac84-49f9-a487-763279a6f658",
  "location": {
    "x": 3737.212646484375,
    "y": 2311.91357421875
  },
  "original_filename": "",
  "parent_id": "34f8909a-6736-49f9-9027-7f7cab5c1acf",
  "pinned": false,
  "playback_position": 0,
  "playback_state": "STOPPED",
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "A video",
  "widget_type": "Video"
}
```

## Download Video

Downloads a single video.

```bash
GET /canvases/:id/videos/:video_id/download
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `video_id` (path)   | uuid    | yes      | ID of the video to download   |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a8b5ed9e-6e57-41c8-96cb-be4aec4bfd03/videos/105a8486-ac84-49f9-a487-763279a6f658/download
```

## Create Video

Creates a video. The request must be a multipart POST request with an optional `json` and mandatory `data` part.

```bash
POST /canvases/:id/videos
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `depth`             | number  | no       | Depth of the video compared to its siblings |
| `location`          | object  | no       | Location of the video relative to its parent |
| `parent_id`         | uuid    | no       | ID of the video's parent       |
| `pinned`            | boolean | no       | Is the video pinned or not     |
| `scale`             | number  | no       | Scale of the video             |
| `size`              | object  | no       | Size of the video              |
| `original_filename` | string  | no       | Original filename of the video |
| `title`             | string  | no       | Title of the video             |
| `playback_state`    | string  | no       | `stopped`, `playing` or `paused` |
| `playback_position` | number  | no       | Current playback position in seconds |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" -F 'json={"title":"A video"}' -F 'data=@SampleVideo_1280x720_1mb.mp4' https://canvus.example.com/api/v1/canvases/a8b5ed9e-6e57-41c8-96cb-be4aec4bfd03/videos
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "",
  "id": "370b112b-c6e9-440f-bdad-f6d6cf6a6d9f",
  "location": {
    "x": 5148.7529296875,
    "y": 2807.719970703125
  },
  "original_filename": "",
  "parent_id": "34f8909a-6736-49f9-9027-7f7cab5c1acf",
  "pinned": false,
  "playback_position": 0,
  "playback_state": "STOPPED",
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "Sample",
  "widget_type": "Video"
}
```

## Update Video

Updates a video.

```bash
PATCH /canvases/:id/videos/:video_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `video_id` (path)   | uuid    | yes      | ID of the video to update     |
| `depth`             | number  | no       | Depth of the video compared to its siblings |
| `location`          | object  | no       | Location of the video relative to its parent |
| `parent_id`         | uuid    | no       | ID of the video's parent       |
| `pinned`            | boolean | no       | Is the video pinned or not     |
| `scale`             | number  | no       | Scale of the video             |
| `size`              | object  | no       | Size of the video              |
| `original_filename` | string  | no       | Original filename of the video |
| `title`             | string  | no       | Title of the video             |
| `playback_state`    | string  | no       | `stopped`, `playing` or `paused` |
| `playback_position` | number  | no       | Current playback position in seconds |

**Example cURL Request**:
```bash
curl -X PATCH -H "Private-Token: <access token>" -d '{"title":"Sample"}' https://canvus.example.com/api/v1/canvases/a8b5ed9e-6e57-41c8-96cb-be4aec4bfd03/videos/105a8486-ac84-49f9-a487-763279a6f658
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "863b0937ee95",
  "id": "105a8486-ac84-49f9-a487-763279a6f658",
  "location": {
    "x": 3737.212646484375,
    "y": 2311.91357421875
  },
  "original_filename": "",
  "parent_id": "34f8909a-6736-49f9-9027-7f7cab5c1acf",
  "pinned": false,
  "playback_position": 0,
  "playback_state": "STOPPED",
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
 

 },
  "state": "normal",
  "title": "Sample",
  "widget_type": "Video"
}
```

## Delete Video

Deletes a video.

```bash
DELETE /canvases/:id/videos/:video_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `video_id` (path)   | uuid    | yes      | ID of the video to delete     |

**Example cURL Request**:
```bash
curl -X DELETE -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a8b5ed9e-6e57-41c8-96cb-be4aec4bfd03/videos/105a8486-ac84-49f9-a487-763279a6f658
```