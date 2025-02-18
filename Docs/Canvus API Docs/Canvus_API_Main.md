
# Canvus API

The **Canvus API** is a RESTful interface enabling automation and content manipulation in Canvus from external applications. It supports actions like creating, reading, updating, and deleting (CRUD) various resources, including canvases, widgets, and user sessions.

## Compatibility

The API follows semantic versioning using a single major version number. Backward-incompatible changes trigger a version number increment. Minor updates introducing new features do not require a version update but can be identified via the `/server-info` endpoint.

**Current Version:** v1.2 (introduced in Canvus 3.1)

## Basic Usage

All API requests should be prefixed with `/api` followed by the major version number. For example:
```bash
curl https://canvus.example.com/api/v1/server-info
```
The API uses **JSON** for data serialization.

## Authentication

Authentication is required for most API endpoints, except for specific ones like `/server-info`. Requests without proper authentication return a `401 Unauthorized` error.

### Access Tokens

Access tokens must be provided in the request header as `Private-Token`. Tokens can be obtained via:
1. **Login endpoint**: Issues a temporary token valid for 24 hours.
2. **Access Token Management**: Permanent tokens created via the Canvus UI or `/access-tokens` endpoint.

**Example**:
```bash
curl --header "Private-Token: <access_token>" https://canvus.example.com/api/v1/canvases
```

## Status Codes

API responses return standard HTTP status codes. Below is an overview of typical codes for successful operations:

| Request Type | Description                                    |
|--------------|------------------------------------------------|
| `GET`        | Returns `200 OK` with the requested resource.  |
| `POST`       | Returns `200 OK` with the created resource.    |
| `PATCH`      | Returns `200 OK` with the updated resource.    |
| `DELETE`     | Returns `200 OK` indicating successful deletion.|

## Error Reporting

When an error occurs, the API responds with a `4xx` status code and a JSON message describing the issue. Common error cases include:
- Missing or invalid access token
- Missing required attributes (e.g., missing image data)
- Invalid attribute values
- Insufficient user permissions

**Example Error Response**:
```json
{
  "msg": "Permission denied"
}
```

## Streaming

The Canvus API supports real-time streaming for most `GET` requests by including the `subscribe` parameter. The server keeps the connection open indefinitely, sending updates as newline-separated JSON objects.

**Example**:
- A `state` parameter indicates whether a resource is `normal` (active) or `deleted`.
