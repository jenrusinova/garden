import ZoneListEntry from './ZoneListEntry.js';

var ZoneList = ({zones, handleClick, handleTitleClick}) => (
<div className='zone-list'>
 {zones.map((zone) =>
   <ZoneListEntry zone = {zone}
   handleClick = {handleClick}
   handleTitleClick = {handleTitleClick}/>)

 }
</div>

)



export default ZoneList