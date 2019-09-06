export default {
  name: "Library",
  props: ["projects"],
  template: `
  <ul class="library">
  <li v-for="(p,k) in projects">
    <ul class="project">
    <li class="pname">{{ p.name }}</li>
    <li class="puuid">{{ k }}</li>
    <ul class="checkouts">
      <li v-for="(details,host) in p.checkouts">
      <ul class="checkout">
        <li class="chost"> {{ host }} </li>
        <li class="cpath"> {{ details.path }} </li>
        <li class="clast"> {{ details.last | formatDate }} </li>
      </ul>
      </li>
    </ul>
    </ul>
  </li>
  </ul>
  </div>
  `
};
