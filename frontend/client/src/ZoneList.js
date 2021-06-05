import ZoneListEntry from './ZoneListEntry.js';

var ZoneList = (props) => (
<div className='zone-list'>
 {props.zones.map((zone) =>
   <ZoneListEntry zone = {zone} />)

 }
</div>

)



export default ZoneList