DEST=output
NODE=node_modules

pack: garden-webdata.tar.gz

$(NODE):
	npm install

$(DEST):
	mkdir $(DEST)

garden-webdata.tar.gz: $(DEST)/index.html $(DEST)/mustache.js $(DEST)/kitten.js $(DEST)/jquery.js $(DEST)/kitten-base.css
	tar -czf $@ -C $(DEST) $(notdir $^)

all: $(DEST) $(DEST)/index.html $(DEST)/mustache.js $(DEST)/kitten.js $(DEST)/jquery.js $(DEST)/kitten-base.css

$(DEST)/kitten-base.css: css/kitten-base.scss
	sass $< > $@

$(DEST)/index.html: index.pug
	pug < $< > $@

$(DEST)/mustache.js: $(NODE)
	cp node_modules/jquery/dist/jquery.js $@

$(DEST)/kitten.js: js/kitten.js
	cp $< $@

$(NODE)/jquery.js: $(NODE)
	cp node_modules/jquery/dist/jquery.js $@

