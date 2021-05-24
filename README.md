# Toy Sprinkler Controller

This project is simple and very lightweight sprinkler controller for Raspberi Pi (and potentially other systems), which is currently under development.

## Testing Locally

For testing purposes the controller application can be built for the local architecture. Note, that it will not do anything other 
than printing about sprinklers being turned on or off.

The controller app requires:
* Go 1.13+
* GNU make (optional)

The frontend requires:
* sass
* pug
* npm

To build and run locally (given you have all the tools available), you can run:
```
$ make
$ (cd bin; ./geck)
```

This will open a web interface at `localhost:8089`.

## Building for Raspberri Pi

TBD

## Zone model

TBD ...


