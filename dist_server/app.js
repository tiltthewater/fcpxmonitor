import Library from "./library.js";

Vue.prototype.$urls = {
	library: "library",
}

Vue.filter("formatDate", function(value) {
  if (value) {
    return moment.unix(value).fromNow();
  } else {
	return "Just opened";
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
