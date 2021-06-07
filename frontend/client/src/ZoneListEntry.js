

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
      Time:{runTime} min
    </div>
    <div className ='choose-time'>Choose new time:<button id='min-5'>5 min</button><button id='min-10'>10 min</button><button id='min-15'>15 min</button></div>
    </h2>
    </div>
  )



}

export default ZoneListEntry;