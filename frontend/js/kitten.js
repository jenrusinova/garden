const zone_default_values = {
    name: 'unknown',
    is_on: false,
    is_running: false,
    runtime: 0,
    next_run: null,
    run_template : ["5m", "10m", "15m"]
};


class CollectTemplates {
    register(templateObject) {
        const template = templateObject.html();
        Mustache.parse(template);
        this.templates[templateObject.attr("id")] = template;
    }

    constructor(doc) {
        this.templates = {}
        const self = this

        doc.find('template').each(
            function (_, el) {
                self.register($(el))
            });
    }

    render(name, context) {
        if (!this.templates.hasOwnProperty(name)) {
            console.error("Template not found : " + name);
            return null;
        }

        return Mustache.render(this.templates[name], context);
    }
}

const templates = new CollectTemplates($('body'));

const renderDuration = function(nanos) {
    const secs = Math.floor(nanos / 1000000000);

    if (secs > 120) {
        return renderDurationSeconds(secs)
    }

    return secs.toString() + "s"
}

const renderDurationMillis = function(millis) {
    const secs = Math.floor(millis / 1000);

    if (secs > 120) {
        return renderDurationSeconds(secs)
    }

    return secs.toString() + "s"
}

const renderDurationSeconds = function(secs) {
    const mins = Math.floor(secs / 60);

    if (mins > 120) {
        return renderDurationHours(mins)
    }

    return mins.toString() + "m"
}

const renderDurationHours = function(mins) {
    const hh = Math.floor(mins / 60);
    return hh.toString() + "h"
}

class ZonePanel {
    constructor(id, container) {
        this.id = id;
        this.container = container;
        this.element = null;

        jQuery.extend(this, zone_default_values);
    }

    update(zone_data) {
        jQuery.extend(this, zone_data);
        this.render();
        return this;
    }

    ensure_element_exists() {
        const id_base = '#zone_' + this.id;
        this.element = this.container.find(id_base);

        if (this.element.length > 0) {
            return;
        }

        this.container.append(templates.render("zone", this));
        this.element = this.container.find(id_base);

        if (this.element.length === 0) {
            this.element = null;
            return;
        }

        const zone = this.element;

        const ch = function (p) {
            return zone.find(id_base + "-" + p);
        };

        this.ctrl_state = ch("state");
        this.ctrl_runtime = ch("runtime");
        this.ctrl_next_run = ch("next_run");
        this.ctrl_actions = ch("actions");
    }

    render() {
        const self = this
        this.ensure_element_exists();

        if (this.element === null) {
            console.error("Element not created for : " + this);
            return;
        }

        this.ctrl_state.prop("checked", this.is_on);
        this.ctrl_runtime.text(renderDuration(this.runtime));

        if (this.next_run != null) {
            var dt = new Date(this.next_run) - new Date()
            this.ctrl_next_run.text(renderDurationMillis(dt.valueOf()));
        } else {
            this.ctrl_next_run.text("N");
        }

        if (!this.is_running) {
            const buttons = this.run_template.map( function (v) {
                return templates.render("run-button", { runtime : v} );
            }).join('');

            this.ctrl_actions.html(buttons);

            this.ctrl_actions.find("button").each(function (idx, button) {
                $(button).click(function () {
                    console.log("Starting : " + self.id);
                    self.doStart()
                });
            })
        } else {
            this.ctrl_actions.html(templates.render("stop-button"), this);

            this.ctrl_actions.find("button").each(function (idx, button) {
                $(button).click(function () {
                    console.log("Stopping : " + self.id);
                    self.doStop()
                });
            })
        }

        return this;
    }
}


const simpleAction = function(url, controller) {
    jQuery.ajax({
        url: url,
        type: 'get',
        cache: false,
        success: function() { controller.load(); },
        async:true,
    });
}


class Controller{
    constructor(zone_container) {
        this.url = "";
        this.zones = {};
        this.zone_container = zone_container;
    }

    add_zone(zone_obj) {
        const self = this
        var zone = null;

        if (this.zones.hasOwnProperty(zone_obj.id)) {
            zone = this.zones[zone_obj.id];
        } else {
            zone = new ZonePanel(zone_obj.id, this.zone_container);

            zone.doStart = function() {
                simpleAction(self.url + "/start/" + zone_obj.id, self)
            }

            zone.doStop = function() {
                simpleAction(self.url + "/stop/" + zone_obj.id, self)
            }

            this.zones[zone_obj.id] = zone;
        }

        return zone.update(zone_obj);
    }

    process_zones(data) {
        const self = this

        jQuery.each(data.zones, function (_, obj) {
            self.add_zone(obj);
        });
    }

    load() {
        const self = this

        jQuery.ajax({
            url: this.url + "/zone/",
            type: 'get',
            dataType: 'json',
            cache: false,
            success: function(data) { self.process_zones(data); },
            async:true,
        });
    }
}



