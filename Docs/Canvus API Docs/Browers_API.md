# Browsers API

The Browsers API operates on web browsers within a canvas.

## List Browsers

Gets all web browsers of the specified canvas.

```bash
GET /canvases/:id/browsers
```

| Attribute          | Type    | Required | Description             |
|--------------------|---------|----------|-------------------------|
| `id` (path)        | uuid    | yes      | ID of the canvas         |
| `subscribe` (query) | boolean | no       | See Streaming            |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/d098152c-8abf-4ebd-8a33-2fdf3ffe0214/browsers
```

**Example Response**:
```json
[
  {
    "depth": 0,
    "id": "82796808-84e4-4309-8d76-bd0cee281a22",
    "location": {
      "x": 4932.95068359375,
      "y": 2223.86572265625
    },
    "main_frame_scroll_offset": {
      "x": 0,
      "y": 0
    },
    "parent_id": "51e47ebf-bab1-4412-a48a-5f635ed22f33",
    "pinned": false,
    "scale": 1,
    "size": {
      "height": 800,
      "width": 1200
    },
    "state": "normal",
    "title": "",
    "transparent_mode": false,
    "url": "https://www.google.com/",
    "widget_type": "Browser"
  }
]
```

## Single Browser

Gets a single browser.

```bash
GET /canvases/:id/browsers/:browser_id
```

| Attribute           | Type    | Required | Description                  |
|---------------------|---------|----------|------------------------------|
| `id` (path)         | uuid    | yes      | ID of the canvas              |
| `browser_id` (path)  | uuid    | yes      | ID of the browser to get       |
| `subscribe` (query) | boolean | no       | See Streaming                 |

**Example cURL Request**:
```bash
curl -H "Private-Token: <access token>" https://canvus.example.com/api/v1/canvases/d098152c-8abf-4ebd-8a33-2fdf3ffe0214/browsers/82796808-84e4-4309-8d76-bd0cee281a22
```

**Example Response**:
```json
{
  "depth": 0,
  "id": "82796808-84e4-4309-8d76-bd0cee281a22",
  "location": {
    "x": 4932.95068359375,
    "y": 2223.86572265625
  },
  "main_frame_scroll_offset": {
    "x": 0,
    "y": 0
  },
  "parent_id": "51e47ebf-bab1-4412-a48a-5f635ed22f33",
  "pinned": false,
  "scale": 1,
  "size": {
    "height": 800,
    "width": 1200
  },
  "state": "normal",
  "title": "",
  "transparent_mode": false,
  "url": "https://www.google.com/",
  "widget_type": "Browser"
}
```
