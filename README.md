Storing JSON data to Mongo DB. The JSON data have to be identified by a valid UUID and sent in the payload as a "uuid" field.

# Usage
```
export DB_NAME=<name>
export DB_ADDRESS=<address>
export USER=<user>
export PASSWORD=<pwd>
./holiday-01
```

# Endpoints

* /store/{collection}/{uuid}
   * GET - Get data from the given collection identified by the given uuid
   * POST - Insert or update data in the given collection based on the given uuid
   
## Examples

* Write request 
```
POST http://localhost:8080/store/holiday/39a42511-5510-4950-ac92-67ec8e9e2f4d

{"uuid":"39a42511-5510-4950-ac92-67ec8e9e2f4d","value":"test2"}
```

* Get request
```
GET http://localhost:8080/store/holiday/39a42511-5510-4950-ac92-67ec8e9e2f4d
```