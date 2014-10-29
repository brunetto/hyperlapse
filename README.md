hyperlapse
==========

Small program to create hyperlapses with Google Street View.

My homemade hyperlapse tool inspired by [http://hyperlapse.tllabs.io/](http://hyperlapse.tllabs.io/).    
It will download (using 4 "threads") the StreetView images corresponding to the data in test.dat and create the gif hyperlapse.

### Installation

If you have a Go installation you can install `hyperlapse` with

````bash
go get github.com/brunetto/hyperlapse
````

otherwise you can just download the binary (for linux)  [here](https://github.com/brunetto/hyperlapse/blob/master/hyperlapse)

### Use

Run it with 

`````bash
./hyperlapse test.dat
````

### Note

The test.dat file contains only one coordinate copied over multiple lines so it will download multiple copies of the same image.

### TODO

* [DONE]Automatically create the gif file with the images.
* Automatically create the list of coordinates given the path

May be useful:  

* https://developers.google.com/maps/documentation/utilities/polylineutility
* http://gpx.cgtk.co.uk/
