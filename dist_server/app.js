import Library from "./library.js";

Vue.prototype.$urls = {
	library: "library",
}

Vue.filter("formatSince", function(value) {
  if (value) {
    return moment.unix(value).fromNow();
  } else {
	return "Opened";
  }
});

Vue.filter("formatTime", function(value) {
  if (value) {
	let m = moment.unix(value)
    return m.format("dddd MMM D @ h:mma");
  }
});

Vue.filter("formatProjectTitle", function(value) {
  if (value) {
    return value.replace(/\.fcpbundle/,"");
  }
});

const App = new Vue({
  el: "#app",
  components: { Library },
  data: { lib: {library:{}} },
  methods: {
    log(line) {
      console.log(line);
    }
  },
	async beforeMount () {
		let response = await fetch(this.$urls.library)
		let data = await response.json()
		this.lib.library = data.library
	},
	created () {
		let self = this
		setInterval(function () {
      		fetch(self.$urls.library)
			.then(function(response) {
				return response.json()
			})
			.then(function(data) {
				self.$set(self.lib, "library", data.library)
			})
		}, 5000); 
	},
  template: `
  <div id="app">
    <Library :projects="lib.library" />
  </div>
  `
});
