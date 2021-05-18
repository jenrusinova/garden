DEST=web

pack: garden-webdata.tar.gz

garden-webdata.tar.gz: $(DEST)/index.html $(DEST)/mustache.js $(DEST)/kitten.js $(DEST)/jquery.js $(DEST)/kitten-base.css
	tar -czf $@ -C $(DEST) $(notdir $^)

all: $(DEST)/index.html $(DEST)/mustache.js $(DEST)/kitten.js $(DEST)/jquery.js $(DEST)/kitten-base.css

$(DEST)/kitten-base.css: css/kitten-base.scss
	sass $< > $@

$(DEST)/index.html: index.pug
	pug < $< > $@

$(DEST)/mustache.js: js/mustache.js
	cp $< $@

$(DEST)/kitten.js: js/kitten.js
	cp $< $@

$(DEST)/jquery.js: js/jquery.js
	cp $< $@


