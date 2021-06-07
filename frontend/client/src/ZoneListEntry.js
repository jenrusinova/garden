

var ZoneListEntry = ({zone, handleClick, handleTitleClick}) => {
  console.log(zone);
  var zoneName = zone.name;
  var isWorking = zone.is_on.toString();
  var runTime = zone.runtime;
  return (
    <div className ='zone-list-entry'>
    <h1><div className = 'zone-list-entry-name'
   >
     {zoneName}
    </div></h1><h5><div className ='title-edit'  onClick={()=>handleTitleClick(zone)
    }>Edit title</div></h5>
    <h2><div className = 'zone-list-entry-working'  onClick={()=> (handleClick(zone))}>
      Active: <button>{isWorking}
     </button>
    </div>
    <div className = 'zone-list-entry-runtime'>
      Time:{runTime}
    </div>
    </h2>
    </div>
  )



}

export default ZoneListEntry;