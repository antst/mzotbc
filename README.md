# Multi-Zone OpenTherm Boiler Controller

There is a typical issue one meets when starts to deal with home automation of heating, 
when multiple zones with TRVs and OpenThem Boiler are involved: calculation of boiler setpoint 
depending on the conditions in the rooms. This software solves it, maybe in simplistic way, 
but it does.

## How does it work:
Assumption is that all communication happens via MQTT. For defined zones, MZOTBC (later "controller")
collects data  from thermal sensors. MQTT topics can be either plain text, or JSON formatted.
It is possible to have multiple thermal sensors per zone. Then weighted average will be used 
(by default weights are 1). For every zone controller also watch for current zone setpoints, also per MQTT.
Additionally, for every zone is defined heating parmeter (either explicitly or default one us used).
Yet another source for the calculation is outside temperature, which is also provided on MQTT topic.
At the end, for every zone boiler setpoint (target water temperature in heating system) is calculated, 
which depends on zone setpoint (target temperature in the zone), current zone temperature, current 
outside temperature and heating parameter for given zone. For calculations, I use some kind of heating curve,
partly semi-empirical partly borrowed via reverse engineering.

## About usage
Code is written in GO. You can run it with `make run`.
You can build your own docker image with provided [Dockerfile](./Dockerfile) or simply with `make docker-build`.
Alternatively you can use docker image available(will be soon) on dockerhub, 
and this is recommended way. Addon for Home Assistant is on the way.
There is a example of [config file](./example_config.yaml), which must be edited for your case and your zones 
and MQTT topics, and renamed to `config.yaml`.

## Code quality and features 
This was originally quick "one evening prototyping" (which works in my home system for quite some time though).
So, don't expect highest quality from the code. It works and that was enough for me :) 
And while it can be more feature-rich (and one can notice I planned it),I never got to the point to add 
features I thought I will. As it turned out controller, the way it is now, is enough.
Although there are a couple of bells and whistles I want to add, but never get to this point, as I need 
them only once a year :) But, if you think you want to make it better or add something - submit PR. 
If you have feature request or find a bug - let me know :)

## Licensing
Code is licensed under terms of [GNU GPLv3 License](./LICENSE).
But if you desperately needed something like this, and you find it covered your neeeds,
maybe can buy me beer one day :)

