# PDFs API

## List PDFs

Gets all PDFs of the specified canvas.

```bash
GET /canvases/:id/pdfs
```

| Attribute          | Type    | Required | Description             |
|--------------------|---------|----------|-------------------------|
| `id` (path)        | uuid    | yes      | ID of the canvas         |
| `subscribe` (query) | boolean | no       | See Streaming            |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a528ddb9-ee56-47f2-a419-0c3a9227996e/pdfs
```

**Example Response**:
```json
[
  {
    "depth": 0,
    "hash": "347b3c308971",
    "id": "d2c53c80-fc5f-4e10-9c8b-28f4a7f06232",
    "index": 1,
    "location": {
      "x": 4629.5146484375,
      "y": 2105.46923828125
    },
    "original_filename": "",
    "parent_id": "0aad755b-c78f-4392-88fb-6efb31d35290",
    "pinned": false,
    "scale": 1,
    "size": {
      "height": 100,
      "width": 100
    },
    "state": "normal",
    "title": "A PDF",
    "widget_type": "Pdf"
  }
]
```

## Single PDF

Gets a single PDF.

```bash
GET /canvases/:id/pdfs/:pdf_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `pdf_id` (path)     | uuid    | yes      | ID of the PDF to get          |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a528ddb9-ee56-47f2-a419-0c3a9227996e/pdfs/d2c53c80-fc5f-4e10-9c8b-28f4a7f06232
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "347b3c308971",
  "id": "d2c53c80-fc5f-4e10-9c8b-28f4a7f06232",
  "index": 1,
  "location": {
    "x": 4629.5146484375,
    "y": 2105.46923828125
  },
  "original_filename": "",
  "parent_id": "0aad755b-c78f-4392-88fb-6efb31d35290",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "A PDF",
  "widget_type": "Pdf"
}
```

## Download PDF

Downloads a single PDF.

```bash
GET /canvases/:id/pdfs/:pdf_id/download
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `pdf_id` (path)     | uuid    | yes      | ID of the PDF to download     |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a528ddb9-ee56-47f2-a419-0c3a9227996e/pdfs/d2c53c80-fc5f-4e10-9c8b-28f4a7f06232/download
```

## Create PDF

Creates a PDF. The request must be a multipart POST with an optional `json` and mandatory `data` part.

```bash
POST /canvases/:id/pdfs
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `depth`             | number  | no       | Depth of the PDF compared to its siblings |
| `location`          | object  | no       | Location of the PDF relative to its parent |
| `parent_id`         | uuid    | no       | ID of the PDF's parent       |
| `pinned`            | boolean | no       | Is the PDF pinned or not     |
| `scale`             | number  | no       | Scale of the PDF             |
| `size`              | object  | no       | Size of the PDF              |
| `original_filename` | string  | no       | Original filename of the PDF |
| `index`             | number  | no       | Currently displayed page of the PDF |
| `title`             | string  | no       | Title of the PDF             |

**Example cURL Request**:
```bash
curl -X POST -H "Private-Token: <access token>" -F 'json={"title":"A PDF"}' -F 'data=@sample.pdf' https://canvus.example.com/api/v1/canvases/a528ddb9-ee56-47f2-a419-0c3a9227996e/pdfs
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "",
  "id": "7456c122-cb58-4493-b66e-2efb706e213a",
  "index": 1,
  "location": {
    "x": 3624.74560546875,
    "y": 2387.43798828125
  },
  "original_filename": "",
  "parent_id": "0aad755b-c78f-4392-88fb-6efb31d35290",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "A PDF",
  "widget_type": "Pdf"
}
```

## Update PDF

Updates a PDF.

```bash
PATCH /canvases/:id/pdfs/:pdf_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `pdf_id` (path)     | uuid    | yes      | ID of the PDF to update       |
| `depth`             | number  | no       | Depth of the PDF compared to its siblings |
| `location`          | object  | no       | Location of the PDF relative to its parent |
| `parent_id`         | uuid    | no       | ID of the PDF's parent       |
| `pinned`            | boolean | no       | Is the PDF pinned or not     |
| `scale`             | number  | no       | Scale of the PDF             |
| `size`              | object  | no       | Size of the PDF              |
| `original_filename` | string  | no       | Original filename of the PDF |
| `index`             | number  | no       | Currently displayed page of the PDF |
| `title`             | string  | no       | Title of the PDF             |

**Example cURL Request**:
```bash
curl -X PATCH -H "Private-Token: <access token>" -d '{"index":2}' https://canvus.example.com/api/v1/canvases/a528ddb9-ee56-47f2-a419-0c3a9227996e/pdfs/d2c53c80-fc5f-4e10-9c8b-28f4a7f06232
```

**Example Response**:
```json
{
  "depth": 0,
  "hash": "347b3c308971",
  "id": "d2c53c80-fc5f-4e10-9c8b-28f4a7f06232",
  "index": 2,
  "location": {
    "x": 4629.5146484375,
    "y": 2105.46923828125
  },
  "original_filename": "",
  "parent_id": "0aad755b-c78f-4392-88fb-6efb31d35290",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 100,
    "width": 100
  },
  "state": "normal",
  "title": "A PDF",
  "widget_type": "Pdf"
}
```

## Delete PDF

Deletes a PDF.

```bash
DELETE /canvases/:id/pdfs/:pdf_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `pdf_id` (path)     | uuid    | yes      | ID of the PDF to delete       |

**Example cURL Request**:
```bash
curl -X DELETE -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/a528ddb9-ee56-47f2-a419-0c3a9227996e/pdfs/d2c53c80-fc5f-4e10-9c8b-28f4a7f06232
```