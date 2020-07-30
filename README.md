# manetheren-serial
The name pays homage to Robert Jordan's epic fantasy series, The Wheel of Time

A serial interface conversion to serve and publish requests for manetheren server data. Works in tandem with [rhuidean-mirror](https://github.com/thehildebreezy/rhuidean-mirror), so far, to pass data using an xbee, but the interface can be extrapolated to any serial interface.

Works as a standalone by accepting and publishing local requets for information over a localhost TCP connection (the server runs on port 5099) to the display which runs as a client (port 5098)

See more about it here [Serial link for a RESTful API in Golang](https://www.tannerjhildebrand.com/serial-services-for-restful-api/)