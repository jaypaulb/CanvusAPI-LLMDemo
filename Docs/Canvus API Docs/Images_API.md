Here is the updated **Images API** with a complete structure based on the previous data:

---

# Images API

## List Images

Gets all images of the specified canvas.

```bash
GET /canvases/:id/images
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/c3d319c2-5eed-456d-acbb-8da46e788157/images
```

**Example Response**:
```json
[
  {
    "depth": 0,
    "hash": "cbcb439a796b",
    "id": "40a30897-57ea-4c35-9bf8-64db7fc710e0",
    "location": {
      "x": 5130.68212890625,
      "y": 2164.9775390625
    },
    "original_filename": "",
    "parent_id": "aad4f204-2324-401e-9acf-257d2dad4d99",
    "pinned": false,
    "scale": 1,
    "size": {
      "height": 100,
      "width": 100
    },
    "state": "normal",
    "title": "A dog",
    "widget_type": "Image"
  }
]
```

## Single Image

Gets a single image.

```bash
GET /canvases/:id/images/:image_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `image_id` (path)   | uuid    | yes      | ID of the image to get        |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/c3d319c2-5eed-456d-acbb-8da46e788157/images/40a30897-57ea-4c35-9bf8-64db7fc710e0
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "cbcb439a796b",
  "id": "40a30897-57ea-4c35-9bf8-64db7fc710e0",
  "location": {
    "x": 5130.68212890625,
    "y": 2164.9775390625
  },
  "original_filename": "",
  "parent_id": "aad4f204-2324-401e-9acf-257d2dad4d99",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "A dog",
  "widget_type": "Image"
}
```

## Create Image

Creates an image.

```bash
POST /canvases/:id/images
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `depth`             | number  | no       | Depth of the image compared to its siblings |
| `location`          | object  | no       | Location of the image relative to its parent |
| `pinned`            | boolean | no       | Is the image pinned or not    |
| `scale`             | number  | no       | Scale of the image            |
| `size`              | object  | no       | Size of the image             |
| `original_filename` | string  | no       | Original filename of the image |
| `title`             | string  | no       | Title of the image            |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" -F 'json={"title":"A new image"}' -F 'data=@sample.jpg' https://canvus.example.com/api/v1/canvases/c3d319c2-5eed-456d-acbb-8da46e788157/images
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "",
  "id": "7456c122-cb58-4493-b66e-2efb706e213a",
  "location": {
    "x": 3624.74560546875,
    "y": 2387.43798828125
  },
  "original_filename": "sample.jpg",
  "parent_id": "aad4f204-2324-401e-9acf-257d2dad4d99",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "A new image",
  "widget_type": "Image"
}
```

## Update Image

Updates an image.

```bash
PATCH /canvases/:id/images/:image_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `image_id` (path)   | uuid    | yes      | ID of the image to update     |
| `depth`             | number  | no       | Depth of the image compared to its siblings |
| `location`          | object  | no       | Location of the image relative to its parent |
| `pinned`            | boolean | no       | Is the image pinned or not    |
| `scale`             | number  | no       | Scale of the image            |
| `size`              | object  | no       | Size of the image             |
| `original_filename` | string  | no       | Original filename of the image |
| `title`             | string  | no       | Title of the image            |

**Example cURL Request**:
```bash
curl -X PATCH -H "Private-Token: <access token>" -d '{"title":"Updated image"}' https://canvus.example.com/api/v1/canvases/c3d319c2-5eed-456d-acbb-8da46e788157/images/40a30897-57ea-4c35-9bf8-64db7fc710e0
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "cbcb439a796b",
  "id": "40a30897-57ea-4c35-9bf8-64db7fc710e0",
  "location": {
    "x": 5130.68212890625,
    "y": 2164.9775390625
  },
  "original_filename": "sample.jpg",
  "parent_id": "aad4f204-2324-401e-9acf-257d2dad4d99",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "Updated image",
  "widget_type": "Image"
}
```

## Delete Image

Deletes an image.

```bash
DELETE /canvases/:id/images/:image_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `image_id` (path)   | uuid    | yes      | ID of the image to delete     |

**Example cURL Request**:
```bash
curl -X DELETE -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/c3d319c2-5eed-456d-acbb-8da46e788157/images/40a30897-57ea-4c35-9bf8-64db7fc710e0
```