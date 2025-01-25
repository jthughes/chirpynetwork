# Chirpy Network
This is a [Boot.dev](https://www.boot.dev/) [course project](https://www.boot.dev/courses/learn-http-servers-golang) about learning HTTP Servers, processing JSON data, building APIs and Webhooks, using JWT authentication and working with data in a PostgreSQL database.

## Installation
1. Set up Postgres
2. Use goose to build the database tables based on /sql/schema/.
2. Set the following environment variables:
```sh
DB_URL # Postgres Database URL
PLATFORM # Dev / Prod to deterimine what features to enable
SECRET_KEY # Generate and store a private key used for generating/validating JWT tokens
POLKA_KEY # Secret Key to authenticate the /api/polka/webhooks webhook.
```

## API Endpoints

| Endpoint | Method | Authenticated | Request | Response | Description | Errors |
| -------- | ------ | ------------- | ------- | -------- | ----------- | ------ |
| ``/api/users`` | ``POST`` | ``false`` | ``email: string``<br>``password:string`` |  Status Code: ``201 CREATED`` <br> Body: ``User`` <br>``id: UUID`` <br> ``created_at: time`` <br> ``updated_at: time`` <br> ``email: string`` <br> ``is_chirpy_red: bool`` | Create a new user. | ``500 INTERNAL SERVER ERROR``: Unable to parse input, unable to hash password, unable to create user. |
| ``/api/users`` | ``PUT`` | ``true`` | ``email: string``<br>``password:string`` |  Status Code: ``201 CREATED`` <br> Body: ``User`` <br>``id: UUID`` <br> ``created_at: time`` <br> ``updated_at: time`` <br> ``email: string`` <br> ``is_chirpy_red: bool`` | Update an existing user. | ``401 UNAUTHORIZED``: Invalid Authorization Bearer Token <br> ``500 INTERNAL SERVER ERROR``: Unable to parse input, unable to hash password, unable to update user. |
| ``/api/chirps`` | ``POST`` | ``true`` | ``body: string`` | Status Code: ``201 CREATED`` <br> Body: <br> ``id: UUID`` <br> ``created_at: time`` <br> ``updated_at: time`` <br> ``body: string`` <br> ``user_id: UUID`` | Post a new chirp for a logged-in user. | ``400 BAD REQUEST``: User does not exist, Chirp is longer than 140 characters <br> ``401 UNAUTHORIZED``: Invalid Authorization Bearer Token <br> ``500 INTERNAL SERVER ERROR``: Unable to decode request, unable to create chirp |
| ``/api/chirps`` | ``GET`` | ``false`` | ``None`` | Status Code: ``201 CREATED`` <br> Body: <br> ``id: UUID`` <br> ``created_at: time`` <br> ``updated_at: time`` <br> ``body: string`` <br> ``user_id: UUID`` | Gets all chirps. Can optionally query ``author_id`` to only get chirps by the specified author. Chirps are sorted ascending by time created, but can be optionally sorted descending by a ``sort`` query. | ``400 BAD REQUEST``: Author id does not exist <br> ``404 NOT FOUND``: No chirps found |
| ``/api/chirps/{chirpID}`` | ``GET`` | ``false`` | ``None`` | Status Code: ``201 CREATED`` <br> Body: <br> ``id: UUID`` <br> ``created_at: time`` <br> ``updated_at: time`` <br> ``body: string`` <br> ``user_id: UUID`` | Retrieves a chirp by id. | ``400 BAD REQUEST``: Invalid chirp id <br> ``404 NOT FOUND``: Chirp not found |
| ``/api/chirps/{chirpID}`` | ``DELETE`` | ``true`` | ``None`` | Status Code: ``204 NO CONTENT`` | Deletes the chirp with the provided ``chirpID``. | ``400 BAD REQUEST``: Invalid chirp id <br> ``401 UNAUTHORIZED``: user not logged in <br> ``403 FORBIDDEN``: user not authorized to delete chirp. <br> ``404 NOT FOUND``: Chirp not found <br> ``500 INTERNAL SERVER ERROR``: Unable to delete chirp |
| ``/api/login`` | ``POST`` | ``false`` |``email: string``<br>``password:string`` | Status Code: ``200 OK`` <br> Body: <br>``id: UUID`` <br> ``created_at: time`` <br> ``updated_at: time`` <br> ``email: string`` <br> ``is_chirpy_red: bool`` <br> ``token: string`` <br> ``refresh_token: string`` | Attempts to log in with a email and password. Receives an access token and a refresh token. The access token must be provided as a Bearer token in the Authorization header of any requests requiring authentication. | ``401 UNAUTHORIZED``: Invalid email or password <br> ``500 INTERNAL SERVER ERROR``: Unable to decode request, unable to create access token, unable to create refresh token, unable to store refresh token, unable to send response |
| ``/api/refresh`` | ``POST`` | ``true`` | ``None`` | Status Code: ``200 OK`` <br> Body: <br> ``token: string`` | Given a valid refresh token as a Bearer token in the Authorization header, returns a new access token. | ``401 UNAUTHORIZED``: Invalid refresh token <br> ``500 INTERNAL SERVER ERROR``: Unable to create acces token, unable to send response |
| ``/api/revoke`` | ``POST`` | ``true`` | ``None`` | Status Code: ``204 NO CONTENT`` | Revokes the provided refresh token. | ``404 NOT FOUND``: Valid refresh token not provided, unable to revoke refresh token. |
| ``/api/polka/webhooks`` | ``POST`` | ``true`` | ``event: string`` <br> ``data: struct {user_id: UUID}`` | ``204 NO CONTENT`` | Requires Valid ApiKey token in Authorization header. Sent by Polka server to indicate ``user_id`` has upgraded to Chirpy Red. | ``401 UNAUTHORIZED``: request not authenticated <br> ``404 NOT FOUND``: user not found <br> ``500 INTERNAL SERVER ERROR``: unable to decode request  |

