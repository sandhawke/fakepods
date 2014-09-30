"Fakepods" is a Crosscloud pod server that isn't actually
decentralized.  It's running at http://USERNAME.fakepods.com, or you
can run your own copy. It's good for testing and app development, and
for helping us playing around with different possible APIs.  At some
point it might grow up to be a real boy.

At the moment it keeps all data in RAM.  You can do persistance
manually by dumping the contents before shutdown and then restoring
them on startup.   Kind of an odd hack for now.

It's also crawling with race conditions.  It needs locks on the data
structures.  So do not trust it with your data!

It implements the Simple Crosscloud RESTful Pod Interface (SCRPI,
pronounced "scrappy"), as below.


SCRPI (Unstable)
================

This REST interface for communicating with a user's personal online
database (pod) favors simplicity and does not require any knowledge of
RDF.  To provide for data integration and interoperability between
applications, some additional machinery is required, but within one
application (and assuming there are no accidental name collisions)
this will suffice.

For this documentation, we'll use $pod to stand for the URL of a pod
and $res to stand for the URL of a particular data object (aka
"resource").  For example, $pod might be "http://alice.fakepods.com"
and $res might be "http://alice.fakepods.com/r3423".


GET from $pod
-------------

* get some basic information about the pod, if it exists

POST to $pod
------------

* the requested content will be stored on the pod at a new $res URL
* if successful, 201 response will include a header, Location: $res, indicating where it was put 
* certain content types have special handling
* for pod data, use application/json, structured as an {...} object, and do not use any key names starting with '@' or '_'.  You may include nested objects, but consider creating them as separate resources and linked to them.  Give the properties reasonable English names without spaces, for now; do not make them be data.

GET from $pod/_active
---------------------

* get a list of all the resources currently on the pod
* structured at ```{ ..., _members=[ { _id=$res1, ... }, { _id=$res2, ... } ] }```

GET from $pod/_nearby
---------------------

* get a list of all the resources available to the pod and relevant to the current operation
* structured the same as $pod/_active
* on a fakepod is simply taken from all the other pods on the same server

GET from $res
-------------

* returns stored content
* if it was an application/json {...} object, certain additional properties will be added, including but not limited to:
** _id the object's canonical URL (basically the same as $res)
** _owner the URL of the pod
** _etag a code indicating this version

PUT to $res  (NOT IMPLEMENTED)
------------------------------

* replaces the content of that object

DELETE $res (NOT IMPLEMENTED)
-----------------------------

* removes the data.   That $res will not be re-assigned as long as this server remembers its state.

Long polling with Wait-For-None-Match
-------------------------------------

* On a GET request, if the header Wait-For-None-Match is present with the value being an etag for the current version, then the server will pause, keeping the connection open until the resource changes to not match that etag.  At that point, processing will proceed normally, as if this header were not present, with the new content being returned.
* The connection might be closed by the server or an intermediate cache.  Some firewalls and proxies close these connections after 60 seconds, so the client should be prepared to re-open it.

Query Parameters
----------------

* Coming soon
