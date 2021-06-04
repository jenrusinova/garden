## Examples

Simple CLI examples of API usage.

Get the list of zones:
```
curl http://localhost:8089/zone/
```

The following APIs are not currently implemented from Web.

Update the name of the zone:
```
curl http://localhost:8089/update/roses/ -H "Content-Type: application/json" -d '{"id" : "roses", "name" : "Front Roses"}'
```

Disable a zone:
```
curl http://localhost:8089/update/roses/ -H "Content-Type: application/json" -d '{"id" : "roses", "is_on" : false}'
```

Start zone for the specified time (in minutes):
```
curl http://localhost:8089/start/lawn?time=1
```

