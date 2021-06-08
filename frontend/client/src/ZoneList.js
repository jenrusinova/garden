import ZoneListEntry from './ZoneListEntry.js';

var ZoneList = ({zones, handleClick, handleTitleClick, handleRuntimeButtonCLick}) => (
<div className='zone-list'>
 {zones.map((zone) =>
   <ZoneListEntry zone = {zone}
   handleClick = {handleClick}
   handleTitleClick = {handleTitleClick}
   handleRuntimeButtonCLick = {handleRuntimeButtonCLick} />)

 }
</div>

)



export default ZoneList