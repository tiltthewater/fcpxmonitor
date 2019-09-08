export default {
  name: "Library",
  props: ["projects"],
  template: `
<ul class="library">
    <li v-for="(p,k) in projects" class="project">
        <ul>
            <li class="pname inlineblock">{{ p.name | formatProjectTitle }}</li>
            <li class="puuid inlineblock">{{ k }}</li>
            <ul class="checkouts">
                <li v-for="(details,host) in p.checkouts">
                    <ul class="checkout">
                        <li class="hostHeader">
                            <div class="chost inlineblock">{{ host }}</div>
                            <div class="clast inlineblock">{{ details.last | formatSince }}</div>
                        </li>
                        <li class="cpath rubik">{{ details.path }}</li>
			            <li v-if="p.info.version">
							<div class="inlineblock rubik">{{ p.info.version }}</div>
							<div class="inlineblock rubik">{{ p.info.version_mtime | formatTime }}</div>
						</li>
                    </ul>
                </li>
            </ul>
        </ul>
    </li>
</ul>
  `
};
