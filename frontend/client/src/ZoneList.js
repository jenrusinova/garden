import ZoneListEntry from './ZoneListEntry.js';

var ZoneList = ({zones, handleClick}) => (
<div className='zone-list'>
 {zones.map((zone) =>
   <ZoneListEntry zone = {zone}
   handleClick = {handleClick} />)

 }
</div>

)



export default ZoneList