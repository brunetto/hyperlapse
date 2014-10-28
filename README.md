hyperlapse
==========

Small program to create hyperlapse with Google Street View.

My homemade tool inspired by [http://hyperlapse.tllabs.io/](http://hyperlapse.tllabs.io/)

### Installation

If you have a Go installation you can install `hyperlapse` with

````bash
go get 
````

otherwise you can just download the binary (for linux)  [here](https://github.com/brunetto/hyperlapse/blob/master/hyperlapse)

### Use

Run it with 

`````bash
./hyperlapse test.dat
````

### Note

The test.dat file contains only one coordinate copied over multiple lines so it will download multiple copies of the same image.



