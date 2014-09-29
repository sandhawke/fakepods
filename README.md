"Fakepods" is a golang crosscloud pod server that isn't actually
decentralized.  It's good for testing and app development, and just
playing around with different possible APIs.  At some point it might
grow up to be a real boy.

At the moment it keeps all data in ram, so everything is lost when the
server restarts.  It's also probably crawling with race conditions.
It's my second golang program, and I'm still trying to understand how
concurrency is best managed.


Simplified Crosscloud Pod Interface ("skippy") UNSTABLE 
=======================================================

This REST interface for communicating with a user's personal online
database (pod) favors simplicity and does not require any knowledge of
RDF.  To provide for data integration and interoperability between
applications, some additional machinery is required, but within one
application (and assuming there are no accidental name collisions)
this will suffice.

For this documentation, we'll use $pod to stand for the URL of a pod
and $res to stand for the URL of a particular data object (aka
"resource").  For example, $pod might be "http://alice.fakepods.com"
and $obj might be "http://alice.fakepods.com/r3423".  (This simplified
interface does not allow applications to choose URLs.)

Single-Resource Operations
--------------------------

### POST to $pod

* the requested content will be stored on the pod at a new $res URL
* if successful, 201 response will include a header, Location: $res, indicating where it was put 
* certain content types have special handling
* for pod data, use application/json, structured as an {...} object, and do not use any key names starting with '@' or '_'.  You may include nested objects, but consider creating them as separate resources and linked to them


### PUT to $res  (NOT IMPLEMENTED)

* replaces the content of that object

### GET from $res

* returns stored content
* if it was an application/json {...} object, certain additional properties will be added, including but not limited to:
** _owner the URL of the pod
** _version a numeric incrementing value
** _id the object's canonical URL

### DELETE $res (NOT IMPLEMENTED)

* removes the data.   That $res will not be reused.

Data Query Operations
---------------------

For these operations, the request must include the headers: "Accept: application/json" and the response will always have "Content-Type: application/json".

### GET $pod/*

* returns all json object stored in this pod
* structured as { "_version": ..., "resources":[ {...}, {...}, ... ], ... }

### GET $pod/**

* like * except related available objects in other pods are also returned

### GET $pod ... ?args

* used to query for a subset of the data and control operations
* properties=[p1,p2,....] only include the given properties (NOT IMPLEMENTED)
* match={p1:val, p2:val, ...} only include matching objects (like mongodb) (NOT IMPLEMENTED)
* wait-for-version-after=v -- response will be delayed until there is a version after v (longpoll)
* after-version=v only include versions after given v (NOT IMPLEMENTED)
* wait-for-changes -- response will be delayed until there is some change in the result (longpoll) (NOT IMPLEMENTED)

