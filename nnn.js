for (var i = 0; i < (1<<17); i++) {
	(function(i) {
		process.nextTick(function() {
			console.log(i);
		})
	})(i)
}

console.log('ok');
